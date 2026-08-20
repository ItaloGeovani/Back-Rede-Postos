package repositorios

import (
	"errors"
	"time"
)

var ErrCombustivelRedeNaoEncontrado = errors.New("combustivel nao encontrado nesta rede")

// CombustivelRedeRegistro linha em rede_combustiveis (escopo por posto).
type CombustivelRedeRegistro struct {
	ID            string    `json:"id"`
	RedeID        string    `json:"id_rede"`
	PostoID       string    `json:"id_posto"`
	Nome          string    `json:"nome"`
	Codigo        string    `json:"codigo"`
	Descricao     string    `json:"descricao"`
	PrecoPorLitro float64   `json:"preco_por_litro"`
	Ativo         bool      `json:"ativo"`
	Ordem         int       `json:"ordem"`
	CriadoEm      time.Time `json:"criado_em"`
	AtualizadoEm  time.Time `json:"atualizado_em"`
}

// CombustivelRedeRepositorio CRUD de combustíveis por posto.
type CombustivelRedeRepositorio interface {
	// ListarPorRede lista combustíveis da rede. Se postoID != "", filtra pelo posto.
	ListarPorRede(redeID, postoID string) ([]*CombustivelRedeRegistro, error)
	BuscarPorID(id, redeID string) (*CombustivelRedeRegistro, error)
	Criar(x *CombustivelRedeRegistro) error
	Atualizar(id, redeID string, atualizar func(*CombustivelRedeRegistro) error) (*CombustivelRedeRegistro, error)
	Excluir(id, redeID string) error
	// ExpandirIDsMesmoTipo, para cada ID, inclui todos os combustíveis da rede com mesmo código
	// (ou mesmo nome se código vazio). Usado em campanhas de escopo rede.
	ExpandirIDsMesmoTipo(redeID string, ids []string) ([]string, error)
}
