package erede

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type oauthToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

var oauthCache sync.Map // key: pv|secret|ambiente -> *oauthToken

type oauthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// ObterBearerToken client_credentials com cache em memória (~20 min).
func ObterBearerToken(ctx context.Context, pv, clientSecret, ambiente string) (string, error) {
	pv = strings.TrimSpace(pv)
	clientSecret = strings.TrimSpace(clientSecret)
	if pv == "" || clientSecret == "" {
		return "", fmt.Errorf("pv e client_secret obrigatorios")
	}
	key := pv + "|" + clientSecret + "|" + ambiente
	if v, ok := oauthCache.Load(key); ok {
		t := v.(*oauthToken)
		if time.Now().Before(t.ExpiresAt.Add(-60 * time.Second)) {
			return t.AccessToken, nil
		}
	}
	tok, err := solicitarToken(ctx, pv, clientSecret, ambiente)
	if err != nil {
		return "", err
	}
	oauthCache.Store(key, tok)
	return tok.AccessToken, nil
}

func solicitarToken(ctx context.Context, pv, clientSecret, ambiente string) (*oauthToken, error) {
	endpoint := oauthURL(ambiente)
	body := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	basic := base64.StdEncoding.EncodeToString([]byte(pv + ":" + clientSecret))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth e.rede status %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var out oauthResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, fmt.Errorf("oauth e.rede sem access_token")
	}
	ttl := out.ExpiresIn
	if ttl <= 0 {
		ttl = 1440
	}
	return &oauthToken{
		AccessToken: strings.TrimSpace(out.AccessToken),
		ExpiresAt:   time.Now().Add(time.Duration(ttl) * time.Second),
	}, nil
}

func oauthURL(ambiente string) string {
	if strings.EqualFold(ambiente, "producao") {
		return "https://api.userede.com.br/redelabs/oauth2/token"
	}
	return "https://rl7-sandbox-api.useredecloud.com.br/oauth2/token"
}
