package erede

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// PixCriado resultado de criar QR Code Pix.
type PixCriado struct {
	TID          string
	QrCodeData   string
	QrCodeImage  string
	ReturnCode   string
	ReturnMessage string
	Reference    string
}

type pixCreateRequest struct {
	Kind      string        `json:"kind"`
	Reference string        `json:"reference"`
	Amount    string        `json:"amount"`
	QrCode    pixQrCodeReq  `json:"qrCode"`
}

type pixQrCodeReq struct {
	DateTimeExpiration string `json:"dateTimeExpiration"`
}

type pixCreateResponse struct {
	Reference string `json:"reference"`
	TID       string `json:"tid"`
	ReturnCode string `json:"returnCode"`
	ReturnMessage string `json:"returnMessage"`
	QrCodeResponse struct {
		QrCodeData  string `json:"qrCodeData"`
		QrCodeImage string `json:"qrCodeImage"`
	} `json:"qrCodeResponse"`
}

// CriarPixQR gera cobrança PIX (valor em reais → centavos sem separador).
func CriarPixQR(ctx context.Context, pv, clientSecret, ambiente string, valorReais float64, reference string, expira time.Time) (*PixCriado, error) {
	if valorReais < 1.0 {
		return nil, fmt.Errorf("valor minimo R$ 1,00")
	}
	reference = strings.TrimSpace(reference)
	if len(reference) > 16 {
		reference = reference[len(reference)-16:]
	}
	if reference == "" {
		return nil, fmt.Errorf("reference obrigatoria")
	}
	centavos := int64(math.Round(valorReais * 100))
	body := pixCreateRequest{
		Kind:      "pix",
		Reference: reference,
		Amount:    fmt.Sprintf("%d", centavos),
		QrCode: pixQrCodeReq{
			DateTimeExpiration: expira.Format("2006-01-02T15:04:05"),
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bearer, err := ObterBearerToken(ctx, pv, clientSecret, ambiente)
	if err != nil {
		return nil, err
	}
	url := txURL(ambiente)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("e.rede criar pix status %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var out pixCreateResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	rc := strings.TrimSpace(out.ReturnCode)
	if rc != "" && rc != "00" {
		return nil, fmt.Errorf("e.rede pix recusado: %s %s", rc, out.ReturnMessage)
	}
	return &PixCriado{
		TID:           strings.TrimSpace(out.TID),
		QrCodeData:    strings.TrimSpace(out.QrCodeResponse.QrCodeData),
		QrCodeImage:   strings.TrimSpace(out.QrCodeResponse.QrCodeImage),
		ReturnCode:    rc,
		ReturnMessage: strings.TrimSpace(out.ReturnMessage),
		Reference:     strings.TrimSpace(out.Reference),
	}, nil
}

func txURL(ambiente string) string {
	if strings.EqualFold(ambiente, "producao") {
		return "https://api.userede.com.br/erede/v2/transactions"
	}
	return "https://sandbox-erede.useredecloud.com.br/v2/transactions"
}
