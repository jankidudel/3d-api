package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	MaxContactMessagesPerIPDay = 10
	MaxContactMessagesPerDay   = 1000
)

var (
	ErrContactDailyLimited  = errors.New("contact daily rate limit exceeded")
	ErrContactGlobalLimited = errors.New("contact global daily rate limit exceeded")
)

type ContactStore interface {
	Increment(context.Context, string, int, time.Time) error
	Decrement(context.Context, string) error
}

type ContactReservation struct {
	keys []string
}

type ContactLimiter struct {
	store  ContactStore
	secret []byte
	now    func() time.Time
}

func NewContactLimiter(store ContactStore, secret string) *ContactLimiter {
	return &ContactLimiter{store: store, secret: []byte(secret), now: time.Now}
}

func (l *ContactLimiter) Reserve(ctx context.Context, remoteAddress string) (*ContactReservation, error) {
	now := l.now().UTC()
	identifier := hmacIdentifier(l.secret, normalizedIP(remoteAddress))
	reservation := &ContactReservation{keys: []string{
		fmt.Sprintf("contact#ip#%s#day#%s", identifier, now.Format("20060102")),
		fmt.Sprintf("contact#global#day#%s", now.Format("20060102")),
	}}
	limits := []int{MaxContactMessagesPerIPDay, MaxContactMessagesPerDay}
	expiresAt := []time.Time{now.Add(48 * time.Hour), now.Add(48 * time.Hour)}
	limitErrors := []error{ErrContactDailyLimited, ErrContactGlobalLimited}
	for index, key := range reservation.keys {
		if err := l.store.Increment(ctx, key, limits[index], expiresAt[index]); err != nil {
			_ = l.Release(ctx, &ContactReservation{keys: reservation.keys[:index]})
			return nil, contactLimitError(err, limitErrors[index])
		}
	}
	return reservation, nil
}

func (l *ContactLimiter) Release(ctx context.Context, reservation *ContactReservation) error {
	if reservation == nil {
		return nil
	}
	for _, key := range reservation.keys {
		if err := l.store.Decrement(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func contactLimitError(err error, limitError error) error {
	if errors.Is(err, ErrLimited) {
		return limitError
	}
	return err
}
