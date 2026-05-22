package erede

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type transactionResponse struct {
	TID        string `json:"tid"`
	Reference  string `json:"reference"`
	ReturnCode string `json:"returnCode"`
	Status     string `json:"status"`
	Kind       string `json:"kind"`
}

// ConsultarTransacaoPorTID GET /v2/transactions/{tid}
func ConsultarTransacaoPorTID(ctx context.Context, pv, clientSecret, ambiente, tid string) (*transactionResponse, error) {
	tid = strings.TrimSpace(tid)
	if tid == "" {
		return nil, fmt.Errorf("tid vazio")
	}
	bearer, err := ObterBearerToken(ctx, pv, clientSecret, ambiente)
	if err != nil {
		return nil, err
	}
	url := txURL(ambiente) + "/" + tid
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("e.rede consulta status %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var out transactionResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TransacaoAprovadaPix heurística: returnCode 00 ou status indicando pago.
func TransacaoAprovadaPix(tx *transactionResponse) bool {
	if tx == nil {
		return false
	}
	if strings.TrimSpace(tx.ReturnCode) == "00" {
		return true
	}
	s := strings.ToUpper(strings.TrimSpace(tx.Status))
	return s == "APPROVED" || s == "PAID" || s == "CONFIRMED"
}
