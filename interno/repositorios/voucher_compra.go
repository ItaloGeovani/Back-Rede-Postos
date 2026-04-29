package repositorios

import (
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

var ErrVoucherCompraNaoEncontrado = errors.New("voucher compra nao encontrado")

// TipoCompraVoucher como o frentista deve honrar o resgate no posto (litros, valor em R$, ou unidade de campanha).
func TipoCompraVoucher(litros *float64, campanhaBaseDesconto string) string {
	if litros != nil && *litros > 1e-9 {
		return "LITRO"
	}
	switch strings.TrimSpace(campanhaBaseDesconto) {
	case modelos.BaseDescontoLitro:
		return "LITRO"
	case modelos.BaseDescontoUnidade:
		return "UNIDADE"
	default:
		return "VALOR"
	}
}

// VoucherCompraRegistro linha de voucher_compras.
type VoucherCompraRegistro struct {
	ID                  string     `json:"id"`
	RedeID              string     `json:"rede_id"`
	UsuarioID           string     `json:"usuario_id"`
	CampanhaID          *string    `json:"id_campanha,omitempty"`
	ValorSolicitado     float64    `json:"valor_solicitado"`
	DescontoAplicado    float64    `json:"desconto_aplicado"`
	ValorFinal          float64    `json:"valor_final"`
	Litros              *float64   `json:"litros,omitempty"`
	CombustivelRedeID   *string    `json:"id_combustivel_rede,omitempty"`
	CombustivelRedeNome string     `json:"combustivel_rede_nome,omitempty"`
	Status              string     `json:"status"`
	MpPaymentID         *int64     `json:"mp_payment_id,omitempty"`
	ReferenciaPagamento *string   `json:"referencia_pagamento,omitempty"`
	CodigoResgate       *string    `json:"codigo_resgate,omitempty"`
	ExpiraPagamento     *time.Time `json:"expira_pagamento_em,omitempty"`
	ExpiraResgate       *time.Time `json:"expira_resgate_em,omitempty"`
	UsadoEm             *time.Time `json:"usado_em,omitempty"`
	PostoUsoID          *string    `json:"id_posto_uso,omitempty"`
	PostoUsoNome        string     `json:"posto_uso_nome,omitempty"`
	OperadorUsuarioID   *string    `json:"operador_usuario_id,omitempty"`
	OperadorPapel       string     `json:"operador_papel,omitempty"`
	OperadorNomeSnapshot string    `json:"operador_nome_snapshot,omitempty"`
	CriadoEm            time.Time  `json:"criado_em"`
	AtualizadoEm        time.Time  `json:"atualizado_em"`
	TipoCompra          string     `json:"tipo_compra,omitempty"`               // LITRO | VALOR | UNIDADE (preenchido quando há JOIN campanhas)
	CampanhaTitulo      string     `json:"campanha_titulo,omitempty"`         // título amigável da campanha, se houver
}

// VoucherCompraConsultaEquipe linha de voucher + cliente dono (consulta frentista/gerente na rede).
type VoucherCompraConsultaEquipe struct {
	VoucherCompraRegistro
	ClienteNomeCompleto string `json:"cliente_nome_completo"`
	ClienteEmail        string `json:"cliente_email,omitempty"`
}

// VoucherCompraPainelLinha voucher + cliente e posto de uso (listagem no painel da rede).
type VoucherCompraPainelLinha struct {
	ID                  string     `json:"id"`
	UsuarioID           string     `json:"usuario_id"`
	CampanhaID          *string    `json:"id_campanha,omitempty"`
	ValorSolicitado     float64    `json:"valor_solicitado"`
	DescontoAplicado    float64    `json:"desconto_aplicado"`
	ValorFinal          float64    `json:"valor_final"`
	Litros              *float64   `json:"litros,omitempty"`
	Status              string     `json:"status"`
	CodigoResgate       *string    `json:"codigo_resgate,omitempty"`
	ExpiraPagamento     *time.Time `json:"expira_pagamento_em,omitempty"`
	ExpiraResgate       *time.Time `json:"expira_resgate_em,omitempty"`
	UsadoEm             *time.Time `json:"usado_em,omitempty"`
	CriadoEm            time.Time  `json:"criado_em"`
	AtualizadoEm        time.Time  `json:"atualizado_em"`
	ClienteNomeCompleto string     `json:"cliente_nome_completo"`
	PostoUsoNome        string     `json:"posto_uso_nome,omitempty"`
	TipoCompra          string     `json:"tipo_compra"`
	CampanhaTitulo      string     `json:"campanha_titulo,omitempty"`
	CombustivelRedeID   *string    `json:"id_combustivel_rede,omitempty"`
	CombustivelRedeNome string     `json:"combustivel_rede_nome,omitempty"`
	OperadorUsuarioID   *string    `json:"operador_usuario_id,omitempty"`
	OperadorPapel       string     `json:"operador_papel,omitempty"`
	OperadorNomeSnapshot string    `json:"operador_nome_snapshot,omitempty"`
}

// VoucherCompraRepositorio persistência de compras de voucher no app.
type VoucherCompraRepositorio interface {
	// CriarPendenteComPix grava após criação do payment no MP (um único INSERT).
	CriarPendenteComPix(x *VoucherCompraRegistro) error
	BuscarPorID(id, usuarioID, redeID string) (*VoucherCompraRegistro, error)
	ListarDoUsuario(redeID, usuarioID string, limite int) ([]*VoucherCompraRegistro, error)
	ContarUsosCampanhaUsuario(campanhaID, usuarioID, redeID string) (int, error)
	// Contar usos aprovados (status ATIVO ou USADO) por campanha, para o app exibir 1/x.
	ListarUsosAprovadosPorCampanha(redeID, usuarioID string) (map[string]int, error)
	BuscarPorIDRede(id, redeID string) (*VoucherCompraRegistro, error)
	AtivarPagamentoAprovado(id, redeID, codigo string, expiraResgate time.Time) error
	// BuscarPorCodigoResgateConsultaEquipe voucher da rede por código de resgate + dados do cliente (nome/e-mail).
	BuscarPorCodigoResgateConsultaEquipe(codigo, redeID string) (*VoucherCompraConsultaEquipe, error)
	// RegistrarBaixaUso marca ATIVO como USADO com posto e operador (frentista/gerente/gestor).
	RegistrarBaixaUso(idVoucher string, redeID string, idPosto *string, operadorUsuarioID, operadorPapel, operadorNome string) error
	// ListarPainelPorRede listagem paginada para o painel; statusFiltro vazio = todos os status.
	ListarPainelPorRede(redeID string, limite, offset int, statusFiltro string) ([]*VoucherCompraPainelLinha, int, error)
}

var ErrVoucherBaixaNaoPermitida = errors.New("baixa nao permitida neste estado do voucher")

// Filtra campanha elegível (mesma lógica pública + pertence à rede).
func CampanhaElegivelApp(c *modelos.Campanha, idRede string, agora time.Time) bool {
	if c == nil || c.IDRede != idRede {
		return false
	}
	if c.Status != modelos.StatusCampanhaAtiva || !c.ValidaNoApp {
		return false
	}
	if c.VigenciaInicio != nil && agora.Before(*c.VigenciaInicio) {
		return false
	}
	if c.VigenciaFim != nil && agora.After(*c.VigenciaFim) {
		return false
	}
	return true
}
