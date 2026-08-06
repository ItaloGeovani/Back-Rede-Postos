package modelos

import "encoding/json"

const (
	GatewayProvedorMercadoPago = "MERCADO_PAGO"
	GatewayProvedorERede       = "E_REDE"
)

const (
	MeioPagamentoPix          = "PIX"
	MeioPagamentoDinheiro     = "DINHEIRO"
	MeioPagamentoMoedaVirtual = "MOEDA_VIRTUAL"
)

// GatewayMeiosHabilitados meios de pagamento configurados no painel.
type GatewayMeiosHabilitados struct {
	Pix           bool `json:"pix"`
	CartaoCredito bool `json:"cartao_credito"`
	CartaoDebito  bool `json:"cartao_debito"`
	Dinheiro      bool `json:"dinheiro"`
	MoedaVirtual  bool `json:"moeda_virtual"`
}

// MeiosPadrao retorna PIX habilitado e demais desligados.
func MeiosPadrao() GatewayMeiosHabilitados {
	return GatewayMeiosHabilitados{Pix: true, CartaoCredito: false, CartaoDebito: false, Dinheiro: false, MoedaVirtual: false}
}

// TemAlgumMeio true se ao menos um meio estiver ligado.
func (m GatewayMeiosHabilitados) TemAlgumMeio() bool {
	return m.Pix || m.CartaoCredito || m.CartaoDebito || m.Dinheiro || m.MoedaVirtual
}

// IntersecaoMeios retorna só os meios ligados nos dois lados (rede ∩ posto).
func IntersecaoMeios(a, b GatewayMeiosHabilitados) GatewayMeiosHabilitados {
	return GatewayMeiosHabilitados{
		Pix:           a.Pix && b.Pix,
		CartaoCredito: a.CartaoCredito && b.CartaoCredito,
		CartaoDebito:  a.CartaoDebito && b.CartaoDebito,
		Dinheiro:      a.Dinheiro && b.Dinheiro,
		MoedaVirtual:  a.MoedaVirtual && b.MoedaVirtual,
	}
}

// ParseGatewayMeiosJSON interpreta JSONB do banco.
func ParseGatewayMeiosJSON(raw []byte) GatewayMeiosHabilitados {
	out := MeiosPadrao()
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	if !out.TemAlgumMeio() {
		out.Pix = true
	}
	return out
}
