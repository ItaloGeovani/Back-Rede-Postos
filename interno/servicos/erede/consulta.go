package erede

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type authorizationBlock struct {
	TID        string `json:"tid"`
	Reference  string `json:"reference"`
	ReturnCode string `json:"returnCode"`
	Status     string `json:"status"`
	Kind       string `json:"kind"`
}

type transactionResponse struct {
	TID           string              `json:"tid"`
	Reference     string              `json:"reference"`
	ReturnCode    string              `json:"returnCode"`
	Status        string              `json:"status"`
	Kind          string              `json:"kind"`
	Authorization *authorizationBlock `json:"authorization"`
}

func (tx *transactionResponse) blocoPix() *authorizationBlock {
	if tx == nil {
		return nil
	}
	if tx.Authorization != nil {
		return tx.Authorization
	}
	if strings.TrimSpace(tx.TID) != "" || strings.TrimSpace(tx.Status) != "" || strings.TrimSpace(tx.ReturnCode) != "" {
		return &authorizationBlock{
			TID:        tx.TID,
			Reference:  tx.Reference,
			ReturnCode: tx.ReturnCode,
			Status:     tx.Status,
			Kind:       tx.Kind,
		}
	}
	return nil
}

// StatusPixLabel status legível para logs (ex.: Approved, Pending).
func StatusPixLabel(tx *transactionResponse) string {
	b := tx.blocoPix()
	if b == nil {
		return ""
	}
	return strings.TrimSpace(b.Status)
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

// TransacaoAprovadaPix PIX pago: doc e.Rede usa authorization.status=Approved e returnCode=00.
func TransacaoAprovadaPix(tx *transactionResponse) bool {
	b := tx.blocoPix()
	if b == nil {
		return false
	}
	if strings.TrimSpace(b.ReturnCode) == "00" {
		return true
	}
	s := strings.ToUpper(strings.TrimSpace(b.Status))
	return s == "APPROVED" || s == "PAID" || s == "CONFIRMED"
}
