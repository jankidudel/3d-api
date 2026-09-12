package contact

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type CaptchaVerifier interface {
	Verify(context.Context, string) (CaptchaVerification, error)
}

type CaptchaVerification struct {
	Valid        bool
	FailureCodes []string
}

type TurnstileVerifier struct {
	secret string
	client *http.Client
}

func NewTurnstileVerifier(secret string, client *http.Client) *TurnstileVerifier {
	return &TurnstileVerifier{secret: secret, client: client}
}

func (v *TurnstileVerifier) Verify(ctx context.Context, token string) (CaptchaVerification, error) {
	form := url.Values{"secret": {v.secret}, "response": {token}}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		turnstileVerifyURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return CaptchaVerification{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := v.client.Do(request)
	if err != nil {
		return CaptchaVerification{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return CaptchaVerification{}, nil
	}
	var result struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return CaptchaVerification{}, err
	}
	return CaptchaVerification{Valid: result.Success, FailureCodes: result.ErrorCodes}, nil
}
