package apihttp

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"3d-api/internal/contact"
	"3d-api/internal/subscriptions"
)

type subscriptionSenderStub struct {
	emails []string
}

func (s *subscriptionSenderStub) SendSubscription(_ context.Context, email string) error {
	s.emails = append(s.emails, email)
	return nil
}

type subscriptionStoreStub struct {
	err error
}

func (s subscriptionStoreStub) Subscribe(context.Context, string, string, time.Time) error {
	return s.err
}

func testSubscriptionHandler(store subscriptions.Store, sender contact.SubscriberSender) *SubscriptionHandler {
	return NewSubscriptionHandler(sender, captchaStub{verified: true}, subscriptions.NewService(store, "test-secret"), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestSubscriptionAcceptsOneVerifiedEmail(t *testing.T) {
	sender := &subscriptionSenderStub{}
	handler := testSubscriptionHandler(subscriptionStoreStub{}, sender)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader(`{"email":"Ada@Example.com","turnstileToken":"token"}`))
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusAccepted || len(sender.emails) != 1 || sender.emails[0] != "ada@example.com" {
		t.Errorf("status = %d, emails = %#v, body = %s", response.Code, sender.emails, response.Body.String())
	}
}

func TestSubscriptionDoesNotSendDuplicateEmail(t *testing.T) {
	sender := &subscriptionSenderStub{}
	handler := testSubscriptionHandler(subscriptionStoreStub{err: subscriptions.ErrAlreadySubscribed}, sender)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader(`{"email":"ada@example.com","turnstileToken":"token"}`))
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "already subscribed") || len(sender.emails) != 0 {
		t.Errorf("status = %d, emails = %#v, body = %s", response.Code, sender.emails, response.Body.String())
	}
}

func TestSubscriptionReportsDailyLimit(t *testing.T) {
	handler := testSubscriptionHandler(subscriptionStoreStub{err: subscriptions.ErrDailyLimited}, &subscriptionSenderStub{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader(`{"email":"ada@example.com","turnstileToken":"token"}`))
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusTooManyRequests || !strings.Contains(response.Body.String(), "Request limit per day exceeded.") {
		t.Errorf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestSubscriptionRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"missing email", `{"turnstileToken":"token"}`, "valid email"},
		{"missing CAPTCHA", `{"email":"ada@example.com"}`, "Complete the CAPTCHA"},
		{"unknown field", `{"email":"ada@example.com","turnstileToken":"token","extra":true}`, "valid email"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			handler := testSubscriptionHandler(subscriptionStoreStub{}, &subscriptionSenderStub{})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader(testCase.body))
			response := httptest.NewRecorder()

			handler.Create(response, request)

			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), testCase.want) {
				t.Errorf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestSubscriptionHoneypotDoesNotStoreOrSend(t *testing.T) {
	sender := &subscriptionSenderStub{}
	handler := testSubscriptionHandler(subscriptionStoreStub{err: errors.New("store must not be called")}, sender)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader(`{"email":"ada@example.com","website":"spam.example","turnstileToken":"token"}`))
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusAccepted || len(sender.emails) != 0 {
		t.Errorf("status = %d, emails = %#v", response.Code, sender.emails)
	}
}
