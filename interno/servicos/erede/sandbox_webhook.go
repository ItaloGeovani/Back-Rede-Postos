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

// Payload conforme doc e.Rede sandbox: POST v1/transactions/notification-URL, campo "URL".
type sandboxNotificationURLRequest struct {
	URL           string                         `json:"URL"`
	Authorization *sandboxNotificationURLAuth `json:"authorization,omitempty"`
}

type sandboxNotificationURLAuth struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type sandboxNotificationURLResponse struct {
	ReturnCode    string `json:"returnCode"`
	ReturnMessage string `json:"returnMessage"`
}

// RegistrarWebhookSandbox cadastra URL de callback Pix no PV de teste (sandbox).
// Doc: https://developer.userede.com.br/e-rede — Simulação de notificação via webhook.
func RegistrarWebhookSandbox(ctx context.Context, pv, clientSecret, ambiente, callbackURL, authType, authToken string) error {
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" {
		return fmt.Errorf("url callback vazia")
	}
	if !strings.EqualFold(ambiente, "sandbox") {
		return fmt.Errorf("registro automatico de webhook so em sandbox")
	}
	var body sandboxNotificationURLRequest
	body.URL = callbackURL
	authType = strings.TrimSpace(authType)
	authToken = strings.TrimSpace(authToken)
	if authType != "" && authToken != "" {
		tok := authToken
		low := strings.ToLower(authType)
		if !strings.HasPrefix(strings.ToLower(tok), low+" ") {
			tok = authType + " " + authToken
		}
		body.Authorization = &sandboxNotificationURLAuth{
			Type:  authType,
			Token: tok,
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	bearer, err := ObterBearerToken(ctx, pv, clientSecret, ambiente)
	if err != nil {
		return err
	}
	// Endpoint oficial sandbox (manual PDF): v1/transactions/notification-URL
	url := "https://sandbox-erede.useredecloud.com.br/v1/transactions/notification-URL"
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
	var out sandboxNotificationURLResponse
	_ = json.Unmarshal(b, &out)
	rc := strings.TrimSpace(out.ReturnCode)
	if rc != "" && rc != "00" {
		return fmt.Errorf("registrar webhook sandbox returnCode %s: %s", rc, strings.TrimSpace(out.ReturnMessage))
	}
	return nil
}
