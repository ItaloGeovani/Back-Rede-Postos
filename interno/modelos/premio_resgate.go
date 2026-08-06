package modelos

import "time"

const (
	PremioResgateAguardandoRetirada = "AGUARDANDO_RETIRADA"
	PremioResgateEntregue           = "ENTREGUE"
	PremioResgateCancelado          = "CANCELADO"
)

// PremioResgate pedido de retirada de premio do catalogo.
type PremioResgate struct {
	ID                   string     `json:"id"`
	IDRede               string     `json:"id_rede"`
	PremioID             string     `json:"premio_id"`
	UsuarioID            string     `json:"usuario_id"`
	TituloSnapshot       string     `json:"titulo_snapshot"`
	ImagemURLSnapshot    string     `json:"imagem_url_snapshot,omitempty"`
	ValorMoeda           float64    `json:"valor_moeda"`
	Status               string     `json:"status"`
	PrazoRetiradaEm      time.Time  `json:"prazo_retirada_em"`
	CriadoEm             time.Time  `json:"criado_em"`
	EntregueEm           *time.Time `json:"entregue_em,omitempty"`
	CanceladoEm          *time.Time `json:"cancelado_em,omitempty"`
	PostoEntregaID       *string    `json:"posto_entrega_id,omitempty"`
	PostoEntregaNome     string     `json:"posto_entrega_nome,omitempty"`
	OperadorUsuarioID    *string    `json:"operador_usuario_id,omitempty"`
	OperadorPapel        string     `json:"operador_papel,omitempty"`
	OperadorNomeSnapshot string     `json:"operador_nome_snapshot,omitempty"`
	MotivoCancelamento   string     `json:"motivo_cancelamento,omitempty"`
	// Campos de listagem painel
	ClienteNomeCompleto string `json:"cliente_nome_completo,omitempty"`
	ClienteEmail        string `json:"cliente_email,omitempty"`
	PrazoVencido        bool   `json:"prazo_vencido,omitempty"`
}
