package servicos

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

// PixCobrancaResult resposta unificada de criação/consulta PIX (MP ou e.Rede).
type PixCobrancaResult struct {
	Provedor             string
	IDExterno            string
	QrCode               string
	QrCodeBase64         string
	Status               string
	GatewayStatusLabel   string // e.Rede: authorization.status (logs)
	Referencia           string
	PaymentIDNumerico    int64 // MP: payment id; e.Rede: 0
}

// CriarPixVoucherInput dados para criar cobrança PIX de voucher.
type CriarPixVoucherInput struct {
	Valor               float64
	Referencia          string
	ExpiraEm            string // RFC3339 para e.Rede; MP ignora e usa notification
	PayerEmail          string
	DocTipo             string
	DocNumero           string
	NotificationURL     string
}

func NormalizarGatewayProvedorAtivo(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == modelos.GatewayProvedorERede {
		return modelos.GatewayProvedorERede
	}
	return modelos.GatewayProvedorMercadoPago
}

func GatewayMeiosToJSON(m modelos.GatewayMeiosHabilitados) ([]byte, error) {
	return json.Marshal(m)
}

// RespostaPixVoucherJSON mantém contrato do app (qr_code, payment_id compatível).
func RespostaPixVoucherJSON(reg *repositorios.VoucherCompraRegistro, pix *PixCobrancaResult) map[string]any {
	out := map[string]any{
		"compra_id":             reg.ID,
		"status":                reg.Status,
		"valor_final":           reg.ValorFinal,
		"tipo_beneficio":        reg.TipoBeneficio,
		"cashback_percentual":   reg.CashbackPercentual,
		"cashback_previsto":     reg.CashbackValor,
		"expira_pagamento":      reg.ExpiraPagamento,
		"qr_code":               "",
		"qr_code_base64":        "",
		"mp_status":             "",
		"gateway_provedor":      reg.GatewayProvedor,
		"gateway_payment_id":    "",
	}
	if pix != nil {
		out["qr_code"] = pix.QrCode
		out["qr_code_base64"] = pix.QrCodeBase64
		out["mp_status"] = pix.Status
		out["gateway_payment_id"] = pix.IDExterno
		if pix.PaymentIDNumerico > 0 {
			out["payment_id"] = pix.PaymentIDNumerico
		} else if id, err := strconv.ParseInt(pix.IDExterno, 10, 64); err == nil {
			out["payment_id"] = id
		} else {
			out["payment_id"] = 0
		}
	}
	anexarMoedaNaResposta(out, reg)
	return out
}

func ValidarMeiosParaProvedor(provedor string, m modelos.GatewayMeiosHabilitados) error {
	provedor = NormalizarGatewayProvedorAtivo(provedor)
	if provedor == modelos.GatewayProvedorERede {
		if m.CartaoCredito || m.CartaoDebito {
			return ErrDadosInvalidos // fase 1: e.Rede só PIX (+ dinheiro offline)
		}
	}
	if !m.TemAlgumMeio() {
		return ErrDadosInvalidos
	}
	return nil
}

// RespostaDinheiroVoucherJSON resposta da compra em dinheiro (código imediato).
func RespostaDinheiroVoucherJSON(reg *repositorios.VoucherCompraRegistro) map[string]any {
	cod := ""
	if reg.CodigoResgate != nil {
		cod = strings.TrimSpace(*reg.CodigoResgate)
	}
	avisoTitulo := "Pague em dinheiro no posto"
	avisoCorpo := "Apresente o código ou QR ao frentista. O voucher só será aprovado após o pagamento em dinheiro."
	restante := ValorRestanteACobrar(reg)
	if UsouMoedaVirtual(reg) && restante > 0 {
		avisoTitulo = "Pague o restante em dinheiro"
		avisoCorpo = fmt.Sprintf(
			"Parte já foi paga com moeda virtual. O frentista deve cobrar apenas R$ %.2f.",
			restante,
		)
	}
	out := map[string]any{
		"compra_id":           reg.ID,
		"status":              reg.Status,
		"meio_pagamento":      reg.MeioPagamento,
		"valor_final":         reg.ValorFinal,
		"tipo_beneficio":      reg.TipoBeneficio,
		"cashback_percentual": reg.CashbackPercentual,
		"cashback_previsto":   reg.CashbackValor,
		"codigo_resgate":      cod,
		"expira_resgate_em":   reg.ExpiraResgate,
		"aviso_titulo":        avisoTitulo,
		"aviso_corpo":         avisoCorpo,
	}
	anexarMoedaNaResposta(out, reg)
	return out
}
