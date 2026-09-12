package subscriptions

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	MaxSubscriptionsPerIPDay = 1
	MaxSubscriptionsPerDay   = 1000
)

var (
	ErrAlreadySubscribed = errors.New("email is already subscribed")
	ErrDailyLimited      = errors.New("subscription daily rate limit exceeded")
	ErrGlobalLimited     = errors.New("subscription global daily rate limit exceeded")
)

type Store interface {
	Subscribe(context.Context, string, string, time.Time) error
}

type Service struct {
	store  Store
	secret []byte
	now    func() time.Time
}

func NewService(store Store, secret string) *Service {
	return &Service{store: store, secret: []byte(secret), now: time.Now}
}

func (s *Service) Subscribe(ctx context.Context, email string, remoteAddress string) error {
	return s.store.Subscribe(
		ctx,
		strings.ToLower(strings.TrimSpace(email)),
		hmacIdentifier(s.secret, normalizedIP(remoteAddress)),
		s.now().UTC(),
	)
}

func hmacIdentifier(secret []byte, value string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func normalizedIP(remoteAddress string) string {
	value := strings.TrimSpace(remoteAddress)
	if host, _, found := strings.Cut(value, ":"); found && !strings.Contains(host, ".") {
		return host
	}
	if host, _, found := strings.Cut(value, ":"); found && strings.Count(value, ":") == 1 {
		return host
	}
	return value
}
