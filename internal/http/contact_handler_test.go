package apihttp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"3d-api/internal/contact"
)

type contactSenderStub struct {
	messages []contact.Message
	err      error
}

func (s *contactSenderStub) Send(_ context.Context, message contact.Message) error {
	s.messages = append(s.messages, message)
	return s.err
}

type captchaStub struct {
	verified bool
	err      error
}

func (s captchaStub) Verify(_ context.Context, _ string) (bool, error) {
	return s.verified, s.err
}

func TestContactAcceptsVerifiedMessageWithoutLoggingContents(t *testing.T) {
	sender := &contactSenderStub{}
	var logs strings.Builder
	handler := NewContactHandler(sender, captchaStub{verified: true}, slog.New(slog.NewTextHandler(&logs, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/contact", strings.NewReader(`{"reason":"Report a problem","name":"Ada","email":"ada@example.com","message":"A message that must not be logged","context":"/stl-quality-check","website":"","turnstileToken":"token"}`))
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusAccepted, response.Body.String())
	}
	if len(sender.messages) != 1 || sender.messages[0].Email != "ada@example.com" {
		t.Fatalf("sent messages = %+v", sender.messages)
	}
	if strings.Contains(logs.String(), "A message that must not be logged") {
		t.Errorf("logs contain message text: %q", logs.String())
	}
}

func TestContactRejectsInvalidOrUnverifiedRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"unknown reason", `{"reason":"Other","email":"ada@example.com","message":"Help","turnstileToken":"token"}`, "Choose a reason"},
		{"missing captcha", `{"reason":"General question","email":"ada@example.com","message":"Help"}`, "Complete the CAPTCHA"},
		{"unverified captcha", `{"reason":"General question","email":"ada@example.com","message":"Help","turnstileToken":"token"}`, "Complete the CAPTCHA"},
		{"unknown field", `{"reason":"General question","email":"ada@example.com","message":"Help","turnstileToken":"token","extra":true}`, "Enter a valid contact message"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			handler := NewContactHandler(&contactSenderStub{}, captchaStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/contact", strings.NewReader(testCase.body))
			response := httptest.NewRecorder()

			handler.Create(response, request)

			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), testCase.want) {
				t.Errorf("status = %d, body = %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestContactHoneypotDoesNotSendEmail(t *testing.T) {
	sender := &contactSenderStub{}
	handler := NewContactHandler(sender, captchaStub{verified: true}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/contact", strings.NewReader(`{"reason":"General question","email":"ada@example.com","message":"Help","website":"spam.example","turnstileToken":"token"}`))
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusAccepted || len(sender.messages) != 0 {
		t.Errorf("status = %d, sent = %d", response.Code, len(sender.messages))
	}
}

func TestContactReportsUnavailableWhenNotConfigured(t *testing.T) {
	handler := NewContactHandler(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/contact", nil)
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
