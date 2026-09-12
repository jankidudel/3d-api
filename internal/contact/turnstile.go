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
	Verify(context.Context, string) (bool, error)
}

type TurnstileVerifier struct {
	secret string
	client *http.Client
}

func NewTurnstileVerifier(secret string, client *http.Client) *TurnstileVerifier {
	return &TurnstileVerifier{secret: secret, client: client}
}

func (v *TurnstileVerifier) Verify(ctx context.Context, token string) (bool, error) {
	form := url.Values{"secret": {v.secret}, "response": {token}}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		turnstileVerifyURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return false, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := v.client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, nil
	}
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Success, nil
}
