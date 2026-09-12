package apihttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"3d-api/internal/contact"
	"3d-api/internal/response"
	"3d-api/internal/subscriptions"
)

const maxSubscriptionRequestBytes = 2 * 1024

type subscriptionRequest struct {
	Email          string `json:"email"`
	Website        string `json:"website"`
	TurnstileToken string `json:"turnstileToken"`
}

type SubscriptionHandler struct {
	sender  contact.SubscriberSender
	captcha contact.CaptchaVerifier
	service *subscriptions.Service
	logger  *slog.Logger
}

func NewSubscriptionHandler(sender contact.SubscriberSender, captcha contact.CaptchaVerifier, service *subscriptions.Service, logger *slog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{sender: sender, captcha: captcha, service: service, logger: logger}
}

func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.sender == nil || h.captcha == nil || h.service == nil {
		response.Error(w, http.StatusServiceUnavailable, "Subscriptions are not available yet. Please check back soon.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSubscriptionRequestBytes)
	var request subscriptionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.Error(w, http.StatusBadRequest, "Enter a valid email address.")
		return
	}
	request.Email = strings.ToLower(strings.TrimSpace(request.Email))
	request.Website = strings.TrimSpace(request.Website)
	request.TurnstileToken = strings.TrimSpace(request.TurnstileToken)
	if message := validateSubscriptionRequest(request); message != "" {
		response.Error(w, http.StatusBadRequest, message)
		return
	}
	if request.Website != "" {
		response.JSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
		return
	}
	verification, err := h.captcha.Verify(r.Context(), request.TurnstileToken)
	if err != nil {
		h.logger.Error("subscription captcha verification failed", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "We could not verify the CAPTCHA. Please try again.")
		return
	}
	if !verification.Valid {
		h.logger.Warn("subscription captcha rejected", "failureCodes", strings.Join(verification.FailureCodes, ","))
		response.Error(w, http.StatusBadRequest, "Complete the CAPTCHA before subscribing.")
		return
	}
	if err := h.service.Subscribe(r.Context(), request.Email, contactClientAddress(r)); err != nil {
		switch {
		case errors.Is(err, subscriptions.ErrAlreadySubscribed):
			response.Error(w, http.StatusConflict, "You've already subscribed.")
		case errors.Is(err, subscriptions.ErrDailyLimited):
			response.Error(w, http.StatusTooManyRequests, "You can subscribe once per day. Please try again tomorrow.")
		case errors.Is(err, subscriptions.ErrGlobalLimited):
			response.Error(w, http.StatusTooManyRequests, "Subscriptions have reached their daily limit. Please try again tomorrow.")
		default:
			h.logger.Error("subscription storage failed", "error", err)
			response.Error(w, http.StatusServiceUnavailable, "We could not save your subscription. Please try again.")
		}
		return
	}
	if err := h.sender.SendSubscription(r.Context(), request.Email); err != nil {
		h.logger.Error("subscription email send failed", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "We could not confirm your subscription. Please try again.")
		return
	}
	response.JSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func validateSubscriptionRequest(request subscriptionRequest) string {
	address, err := mail.ParseAddress(request.Email)
	if err != nil || address.Address != request.Email || len(request.Email) > 254 {
		return "Enter a valid email address."
	}
	if request.Website != "" && len(request.Website) > 240 {
		return "Enter a valid subscription request."
	}
	if request.TurnstileToken == "" {
		return "Complete the CAPTCHA before subscribing."
	}
	return ""
}
