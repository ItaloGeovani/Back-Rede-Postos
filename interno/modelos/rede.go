package modelos

import "time"

const (
	GatewayPagamentoModoRede  = "REDE"
	GatewayPagamentoModoPosto = "POSTO"
)

type Rede struct {
	ID                  string    `json:"id"`
	NomeFantasia        string    `json:"nome_fantasia"`
	RazaoSocial         string    `json:"razao_social"`
	CNPJ                string    `json:"cnpj"`
	EmailContato        string    `json:"email_contato"`
	Telefone            string    `json:"telefone"`
	ValorImplantacao    float64   `json:"valor_implantacao"`
	ValorMensalidade    float64   `json:"valor_mensalidade"`
	PrimeiroCobranca    time.Time `json:"primeiro_cobranca"`
	DiaCobranca         int       `json:"dia_cobranca"`
	MoedaVirtualNome    string    `json:"moeda_virtual_nome"`
	MoedaVirtualCotacao float64   `json:"moeda_virtual_cotacao"`
	// MoedaVirtualExpiraDias: 0 = sem expiração; >0 = dias para cada crédito novo.
	MoedaVirtualExpiraDias int `json:"moeda_virtual_expira_dias"`
	// VoucherDiasValidadeResgate dias para usar o saldo no posto após o PIX aprovado.
	VoucherDiasValidadeResgate int `json:"voucher_dias_validade_resgate"`
	// VoucherMinutosExpiraPagamentoPix tempo para pagar a cobrança PIX antes de expirar.
	VoucherMinutosExpiraPagamentoPix int `json:"voucher_minutos_expira_pagamento_pix"`
	// GatewayPagamentoModo: REDE (conta única) ou POSTO (conta por unidade).
	GatewayPagamentoModo string `json:"gateway_pagamento_modo"`
	// GatewayProvedorAtivo: MERCADO_PAGO ou E_REDE (exclusivo).
	GatewayProvedorAtivo string `json:"gateway_provedor_ativo"`
	// GatewayMeiosHabilitados: pix, cartao_credito, cartao_debito.
	GatewayMeiosHabilitados GatewayMeiosHabilitados `json:"gateway_meios_habilitados"`
	// Módulos opcionais do app (painel: Configuracoes; padrao false).
	AppModuloIndiqueGanhe  bool      `json:"app_modulo_indique_ganhe"`
	AppModuloCheckinDiario bool      `json:"app_modulo_checkin_diario"`
	AppModuloGireGanhe     bool      `json:"app_modulo_gire_ganhe"`
	AppModuloRedesSociais  bool      `json:"app_modulo_redes_sociais"`
	Ativa                  bool      `json:"ativa"`
	CriadoEm               time.Time `json:"criado_em"`
	AtualizadoEm           time.Time `json:"atualizado_em"`
}
