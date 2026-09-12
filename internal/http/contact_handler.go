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
	"3d-api/internal/ratelimit"
	"3d-api/internal/response"
)

const maxContactRequestBytes = 16 * 1024

type contactRequest struct {
	Reason         string `json:"reason"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Message        string `json:"message"`
	Context        string `json:"context"`
	Website        string `json:"website"`
	TurnstileToken string `json:"turnstileToken"`
}

type ContactHandler struct {
	sender  contact.Sender
	captcha contact.CaptchaVerifier
	limiter *ratelimit.ContactLimiter
	logger  *slog.Logger
}

func NewContactHandler(sender contact.Sender, captcha contact.CaptchaVerifier, limiter *ratelimit.ContactLimiter, logger *slog.Logger) *ContactHandler {
	return &ContactHandler{sender: sender, captcha: captcha, limiter: limiter, logger: logger}
}

func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.sender == nil || h.captcha == nil || h.limiter == nil {
		response.Error(w, http.StatusServiceUnavailable, "Contact delivery is being configured. Please check back soon.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxContactRequestBytes)
	var request contactRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.Error(w, http.StatusBadRequest, "Enter a valid contact message.")
		return
	}
	request = trimContactRequest(request)
	if message := validateContactRequest(request); message != "" {
		response.Error(w, http.StatusBadRequest, message)
		return
	}
	if request.Website != "" {
		response.JSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
		return
	}
	verification, err := h.captcha.Verify(r.Context(), request.TurnstileToken)
	if err != nil {
		h.logger.Error("contact captcha verification failed", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "We could not verify the CAPTCHA. Please try again.")
		return
	}
	if !verification.Valid {
		h.logger.Warn("contact captcha rejected", "failureCodes", strings.Join(verification.FailureCodes, ","))
		response.Error(w, http.StatusBadRequest, "Complete the CAPTCHA before sending your message.")
		return
	}
	if err := h.limiter.Allow(r.Context(), contactClientAddress(r)); err != nil {
		switch {
		case errors.Is(err, ratelimit.ErrContactHourlyLimited):
			response.Error(w, http.StatusTooManyRequests, "You have sent the maximum number of messages for this hour. Please try again later.")
		case errors.Is(err, ratelimit.ErrContactDailyLimited):
			response.Error(w, http.StatusTooManyRequests, "You have sent the maximum number of messages for today. Please try again tomorrow.")
		case errors.Is(err, ratelimit.ErrContactGlobalLimited):
			response.Error(w, http.StatusTooManyRequests, "Contact delivery has reached its daily limit. Please try again tomorrow.")
		default:
			h.logger.Error("contact rate limit check failed", "error", err)
			response.Error(w, http.StatusServiceUnavailable, "We could not send your message. Please try again.")
		}
		return
	}
	if err := h.sender.Send(r.Context(), contact.Message{
		Reason:  request.Reason,
		Name:    request.Name,
		Email:   request.Email,
		Body:    request.Message,
		Context: request.Context,
	}); err != nil {
		h.logger.Error("contact email send failed", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "We could not send your message. Please try again.")
		return
	}
	response.JSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func contactClientAddress(request *http.Request) string {
	if forwardedFor := request.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		if clientAddress, _, found := strings.Cut(forwardedFor, ","); found {
			return strings.TrimSpace(clientAddress)
		}
		return strings.TrimSpace(forwardedFor)
	}
	return request.RemoteAddr
}

func trimContactRequest(request contactRequest) contactRequest {
	request.Reason = strings.TrimSpace(request.Reason)
	request.Name = strings.TrimSpace(request.Name)
	request.Email = strings.TrimSpace(request.Email)
	request.Message = strings.TrimSpace(request.Message)
	request.Context = strings.TrimSpace(request.Context)
	request.Website = strings.TrimSpace(request.Website)
	request.TurnstileToken = strings.TrimSpace(request.TurnstileToken)
	return request
}

func validateContactRequest(request contactRequest) string {
	switch request.Reason {
	case "Report a problem", "Request a tool or feature", "General question", "Partnership or business enquiry":
	default:
		return "Choose a reason for getting in touch."
	}
	if request.Name != "" && len(request.Name) > 120 {
		return "Name must be 120 characters or fewer."
	}
	address, err := mail.ParseAddress(request.Email)
	if err != nil || address.Address != request.Email || len(request.Email) > 254 {
		return "Enter a valid email address."
	}
	if request.Message == "" {
		return "Enter a message."
	}
	if len(request.Message) > 5000 {
		return "Message must be 5,000 characters or fewer."
	}
	if len(request.Context) > 240 {
		return "Page or tool must be 240 characters or fewer."
	}
	if request.Website != "" && len(request.Website) > 240 {
		return "Enter a valid contact message."
	}
	if request.TurnstileToken == "" {
		return "Complete the CAPTCHA before sending your message."
	}
	return ""
}
