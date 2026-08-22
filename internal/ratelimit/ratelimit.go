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
)

var ErrLimited = errors.New("rate limit exceeded")

type Window struct {
	Key       string
	Limit     int
	ExpiresAt time.Time
	RetryAt   time.Time
}

type LimitedError struct{ RetryAt time.Time }

func (e *LimitedError) Error() string        { return ErrLimited.Error() }
func (e *LimitedError) Is(target error) bool { return target == ErrLimited }

type Store interface {
	Consume(context.Context, []Window) error
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
	now := l.now().UTC()
	hourEnd := now.Truncate(time.Hour).Add(time.Hour)
	dayEnd := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	id := l.identifier(ip)
	windows := []Window{
		{Key: "hour#" + hourEnd.Format("2006010215") + "#" + id, Limit: MaxAcceptedJobsPerHour, ExpiresAt: hourEnd.Add(time.Hour), RetryAt: hourEnd},
		{Key: "day#" + dayEnd.Format("20060102") + "#" + id, Limit: MaxAcceptedJobsPerDay, ExpiresAt: dayEnd.Add(24 * time.Hour), RetryAt: dayEnd},
	}
	if err := l.store.Consume(ctx, windows); err != nil {
		if errors.Is(err, ErrLimited) {
			retryAt := hourEnd
			var limited *LimitedError
			if errors.As(err, &limited) {
				retryAt = limited.RetryAt
			}
			return retryAt.Sub(now).Round(time.Second), ErrLimited
		}
		return 0, err
	}
	return 0, nil
}

func (l *Limiter) identifier(ip string) string {
	mac := hmac.New(sha256.New, l.secret)
	_, _ = mac.Write([]byte(ip))
	return hex.EncodeToString(mac.Sum(nil))
}
