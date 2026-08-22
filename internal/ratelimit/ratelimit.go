package ratelimit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"time"
)

const (
	MaxAcceptedJobsPerHour = 10
	MaxAcceptedJobsPerDay  = 30
	RollingHour            = 60 * time.Minute
	RollingDay             = 24 * time.Hour
)

var ErrLimited = errors.New("rate limit exceeded")

// Store keeps a bounded (maximum 30) timestamp list per HMAC-derived IP.
type Store interface {
	Consume(context.Context, string, time.Time) error
}
type Limiter struct {
	store  Store
	secret []byte
	now    func() time.Time
}

func New(store Store, secret string) *Limiter {
	return &Limiter{store: store, secret: []byte(secret), now: time.Now}
}

func (l *Limiter) Allow(ctx context.Context, ip string) (time.Duration, error) {
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	if err := l.store.Consume(ctx, l.identifier(ip), l.now().UTC()); err != nil {
		return 0, err
	}
	return 0, nil
}

func (l *Limiter) identifier(ip string) string {
	mac := hmac.New(sha256.New, l.secret)
	_, _ = mac.Write([]byte(ip))
	return hex.EncodeToString(mac.Sum(nil))
}
