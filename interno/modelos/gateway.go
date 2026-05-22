package modelos

import "encoding/json"

const (
	GatewayProvedorMercadoPago = "MERCADO_PAGO"
	GatewayProvedorERede       = "E_REDE"
)

// GatewayMeiosHabilitados meios de pagamento configurados no painel.
type GatewayMeiosHabilitados struct {
	Pix            bool `json:"pix"`
	CartaoCredito  bool `json:"cartao_credito"`
	CartaoDebito   bool `json:"cartao_debito"`
}

// MeiosPadrao retorna PIX habilitado e cartões desligados.
func MeiosPadrao() GatewayMeiosHabilitados {
	return GatewayMeiosHabilitados{Pix: true, CartaoCredito: false, CartaoDebito: false}
}

// ParseGatewayMeiosJSON interpreta JSONB do banco.
func ParseGatewayMeiosJSON(raw []byte) GatewayMeiosHabilitados {
	out := MeiosPadrao()
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	if !out.Pix && !out.CartaoCredito && !out.CartaoDebito {
		out.Pix = true
	}
	return out
}
