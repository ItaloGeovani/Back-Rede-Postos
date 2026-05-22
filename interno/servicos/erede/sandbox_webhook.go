package erede

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type webhookRegisterRequest struct {
	URL           string `json:"url"`
	Authorization struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	} `json:"authorization"`
}

// RegistrarWebhookSandbox cadastra URL de callback no PV de teste (sandbox).
func RegistrarWebhookSandbox(ctx context.Context, pv, clientSecret, ambiente, callbackURL, authType, authToken string) error {
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" {
		return fmt.Errorf("url callback vazia")
	}
	if !strings.EqualFold(ambiente, "sandbox") {
		return fmt.Errorf("registro automatico de webhook so em sandbox")
	}
	// Endpoint de simulação conforme documentação e.Rede (sandbox)
	url := "https://sandbox-erede.useredecloud.com.br/v2/transactions/webhook"
	var body webhookRegisterRequest
	body.URL = callbackURL
	body.Authorization.Type = strings.TrimSpace(authType)
	if body.Authorization.Type == "" {
		body.Authorization.Type = "Bearer"
	}
	body.Authorization.Token = strings.TrimSpace(authToken)
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	bearer, err := ObterBearerToken(ctx, pv, clientSecret, ambiente)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("registrar webhook sandbox status %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}
