package ratelimit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"
)

func HourlyLimitMessage() string {
	return fmt.Sprintf("You can upload up to %d files per hour. Please try again later.", MaxAcceptedJobsPerHour)
}

func DailyLimitMessage() string {
	return fmt.Sprintf("You can upload up to %d files per 24 hours. Please try again later.", MaxAcceptedJobsPerDay)
}

func DailyByteLimitMessage() string {
	return fmt.Sprintf("Your %d MiB 24-hour upload allowance has been reached. Please try again later.", MaxAcceptedBytesPerDay/(1024*1024))
}

const (
	MaxAcceptedJobsPerHour       = 10
	MaxAcceptedJobsPerDay        = 30
	MaxAcceptedBytesPerDay int64 = 256 * 1024 * 1024
	RollingHour                  = 60 * time.Minute
	RollingDay                   = 24 * time.Hour
)

var (
	ErrLimited      = errors.New("rate limit exceeded")
	ErrDailyLimited = errors.New("daily upload limit exceeded")
	ErrByteLimited  = errors.New("upload byte limit exceeded")
)

// Store keeps a bounded (maximum 30) timestamp list per HMAC-derived IP.
type Store interface {
	Consume(context.Context, string, string, int64, time.Time) error
}
type Limiter struct {
	store  Store
	secret []byte
	now    func() time.Time
}

func New(store Store, secret string) *Limiter {
	return &Limiter{store: store, secret: []byte(secret), now: time.Now}
}

func (l *Limiter) Allow(ctx context.Context, identifier string, uploadID string, size int64) (time.Duration, error) {
	if err := l.store.Consume(ctx, identifier, uploadID, size, l.now().UTC()); err != nil {
		return 0, err
	}
	return 0, nil
}

func (l *Limiter) Identifier(ip string) string {
	return hmacIdentifier(l.secret, normalizedIP(ip))
}

func normalizedIP(ip string) string {
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

func hmacIdentifier(secret []byte, ip string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(ip))
	return hex.EncodeToString(mac.Sum(nil))
}
