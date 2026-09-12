package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	MaxContactMessagesPerIPHour = 3
	MaxContactMessagesPerIPDay  = 10
	MaxContactMessagesPerDay    = 100
)

var (
	ErrContactHourlyLimited = errors.New("contact hourly rate limit exceeded")
	ErrContactDailyLimited  = errors.New("contact daily rate limit exceeded")
	ErrContactGlobalLimited = errors.New("contact global daily rate limit exceeded")
)

type ContactStore interface {
	Increment(context.Context, string, int, time.Time) error
}

type ContactLimiter struct {
	store  ContactStore
	secret []byte
	now    func() time.Time
}

func NewContactLimiter(store ContactStore, secret string) *ContactLimiter {
	return &ContactLimiter{store: store, secret: []byte(secret), now: time.Now}
}

func (l *ContactLimiter) Allow(ctx context.Context, remoteAddress string) error {
	now := l.now().UTC()
	identifier := hmacIdentifier(l.secret, normalizedIP(remoteAddress))
	if err := l.store.Increment(ctx, fmt.Sprintf("contact#ip#%s#hour#%s", identifier, now.Format("2006010215")), MaxContactMessagesPerIPHour, now.Add(2*time.Hour)); err != nil {
		return contactLimitError(err, ErrContactHourlyLimited)
	}
	if err := l.store.Increment(ctx, fmt.Sprintf("contact#ip#%s#day#%s", identifier, now.Format("20060102")), MaxContactMessagesPerIPDay, now.Add(48*time.Hour)); err != nil {
		return contactLimitError(err, ErrContactDailyLimited)
	}
	if err := l.store.Increment(ctx, fmt.Sprintf("contact#global#day#%s", now.Format("20060102")), MaxContactMessagesPerDay, now.Add(48*time.Hour)); err != nil {
		return contactLimitError(err, ErrContactGlobalLimited)
	}
	return nil
}

func contactLimitError(err error, limitError error) error {
	if errors.Is(err, ErrLimited) {
		return limitError
	}
	return err
}
