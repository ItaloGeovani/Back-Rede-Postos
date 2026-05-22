package erede

import (
	"encoding/json"
	"errors"
	"strings"
)

// WebhookPayload notificação Pix e.Rede.
type WebhookPayload struct {
	CompanyNumber string   `json:"companyNumber"`
	Events        []string `json:"events"`
	Data          struct {
		ID string `json:"id"`
	} `json:"data"`
}

// ParseWebhook extrai tid e verifica evento de pagamento aprovado.
func ParseWebhook(body []byte) (tid string, aprovado bool, err error) {
	if len(body) == 0 {
		return "", false, errors.New("corpo vazio")
	}
	var p WebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return "", false, err
	}
	tid = strings.TrimSpace(p.Data.ID)
	for _, ev := range p.Events {
		if strings.TrimSpace(ev) == "PV.UPDATE_TRANSACTION_PIX" {
			aprovado = true
			break
		}
	}
	if tid == "" {
		return "", false, errors.New("webhook sem data.id (tid)")
	}
	return tid, aprovado, nil
}
