package modelos

import (
	"encoding/json"
	"time"
)

const (
	EventoVoucherGerado  = "VOUCHER_GERADO"
	EventoVoucherPago    = "VOUCHER_PAGO"
	EventoVoucherBaixa   = "VOUCHER_BAIXA"
	EventoCampanhaCriada = "CAMPANHA_CRIADA"
	EventoCampanhaAtivada = "CAMPANHA_ATIVADA"
)

// EventoOperacional log de negócio da rede.
type EventoOperacional struct {
	ID           string          `json:"id"`
	IDRede       string          `json:"id_rede"`
	IDPosto      *string         `json:"id_posto,omitempty"`
	PostoNome    string          `json:"posto_nome,omitempty"`
	TipoEvento   string          `json:"tipo_evento"`
	EntidadeTipo string          `json:"tipo_entidade"`
	IDEntidade   *string         `json:"id_entidade,omitempty"`
	Titulo       string          `json:"titulo"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	CriadoEm     time.Time       `json:"criado_em"`
	// Compatível com UI de auditoria antiga
	DadosNovos json.RawMessage `json:"dados_novos,omitempty"`
}

// RedeWhatsAppNotificacoes config Evolution por rede.
type RedeWhatsAppNotificacoes struct {
	IDRede              string    `json:"id_rede"`
	Habilitado          bool      `json:"habilitado"`
	InstanceName        string    `json:"instance_name"`
	InstanceToken       string    `json:"instance_token,omitempty"`
	InstanceTokenMasked string    `json:"instance_token_masked,omitempty"`
	GroupJID            string    `json:"group_jid"`
	NotifyVoucherGerado bool      `json:"notify_voucher_gerado"`
	NotifyVoucherPago   bool      `json:"notify_voucher_pago"`
	NotifyVoucherBaixa  bool      `json:"notify_voucher_baixa"`
	NotifyCampanha      bool      `json:"notify_campanha"`
	AtualizadoEm        time.Time `json:"atualizado_em"`
}

func (c *RedeWhatsAppNotificacoes) FlagParaTipo(tipo string) bool {
	if c == nil || !c.Habilitado {
		return false
	}
	switch tipo {
	case EventoVoucherGerado:
		return c.NotifyVoucherGerado
	case EventoVoucherPago:
		return c.NotifyVoucherPago
	case EventoVoucherBaixa:
		return c.NotifyVoucherBaixa
	case EventoCampanhaCriada, EventoCampanhaAtivada:
		return c.NotifyCampanha
	default:
		return false
	}
}
