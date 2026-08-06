package notificacoes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// EvolutionSendText envia texto via Evolution Go POST /send/text.
func EvolutionSendText(ctx context.Context, baseURL, apikey, number, text string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	apikey = strings.TrimSpace(apikey)
	number = strings.TrimSpace(number)
	text = strings.TrimSpace(text)
	if baseURL == "" {
		return fmt.Errorf("evolution base url vazia")
	}
	if apikey == "" {
		return fmt.Errorf("evolution apikey vazia")
	}
	if number == "" {
		return fmt.Errorf("destino whatsapp vazio")
	}
	if text == "" {
		return fmt.Errorf("texto whatsapp vazio")
	}
	body, err := json.Marshal(map[string]any{
		"number":    number,
		"text":      text,
		"formatJid": true,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/send/text", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apikey)

	client := &http.Client{Timeout: 20 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("evolution status %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}
