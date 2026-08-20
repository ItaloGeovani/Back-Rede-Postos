package servicos

import (
	"strings"

	"gaspass-servidor/interno/repositorios"
)

// ServicoCombustivelRede catálogo de combustíveis e preço por litro por posto.
type ServicoCombustivelRede struct {
	repo     repositorios.CombustivelRedeRepositorio
	redeRepo repositorios.RedeRepositorio
}

func NovoServicoCombustivelRede(
	repo repositorios.CombustivelRedeRepositorio,
	rede repositorios.RedeRepositorio,
) *ServicoCombustivelRede {
	return &ServicoCombustivelRede{repo: repo, redeRepo: rede}
}

type CriarCombustivelRedeInput struct {
	IDPosto       string
	Nome          string
	Codigo        string
	Descricao     string
	PrecoPorLitro float64
	Ordem         int
	Ativo         bool
}

type AtualizarCombustivelRedeInput struct {
	ID            string
	Nome          string
	Codigo        string
	Descricao     string
	PrecoPorLitro float64
	Ordem         int
	Ativo         bool
}

func (s *ServicoCombustivelRede) Listar(idRede, idPosto string) ([]*repositorios.CombustivelRedeRegistro, error) {
	idRede = strings.TrimSpace(idRede)
	idPosto = strings.TrimSpace(idPosto)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.redeRepo.BuscarPorID(idRede); err != nil {
		return nil, err
	}
	return s.repo.ListarPorRede(idRede, idPosto)
}

func (s *ServicoCombustivelRede) Criar(idRede string, in CriarCombustivelRedeInput) (*repositorios.CombustivelRedeRegistro, error) {
	idRede = strings.TrimSpace(idRede)
	idPosto := strings.TrimSpace(in.IDPosto)
	if idRede == "" || idPosto == "" {
		return nil, ErrDadosInvalidos
	}
	nome := strings.TrimSpace(in.Nome)
	if nome == "" {
		return nil, ErrDadosInvalidos
	}
	if in.PrecoPorLitro < 0 {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.redeRepo.BuscarPorID(idRede); err != nil {
		return nil, err
	}
	reg := &repositorios.CombustivelRedeRegistro{
		RedeID:        idRede,
		PostoID:       idPosto,
		Nome:          nome,
		Codigo:        strings.TrimSpace(in.Codigo),
		Descricao:     strings.TrimSpace(in.Descricao),
		PrecoPorLitro: in.PrecoPorLitro,
		Ativo:         in.Ativo,
		Ordem:         in.Ordem,
	}
	if err := s.repo.Criar(reg); err != nil {
		return nil, err
	}
	return reg, nil
}

func (s *ServicoCombustivelRede) Atualizar(idRede string, in AtualizarCombustivelRedeInput) (*repositorios.CombustivelRedeRegistro, error) {
	idRede = strings.TrimSpace(idRede)
	id := strings.TrimSpace(in.ID)
	if idRede == "" || id == "" {
		return nil, ErrDadosInvalidos
	}
	nome := strings.TrimSpace(in.Nome)
	if nome == "" {
		return nil, ErrDadosInvalidos
	}
	if in.PrecoPorLitro < 0 {
		return nil, ErrDadosInvalidos
	}
	return s.repo.Atualizar(id, idRede, func(r *repositorios.CombustivelRedeRegistro) error {
		r.Nome = nome
		r.Codigo = strings.TrimSpace(in.Codigo)
		r.Descricao = strings.TrimSpace(in.Descricao)
		r.PrecoPorLitro = in.PrecoPorLitro
		r.Ordem = in.Ordem
		r.Ativo = in.Ativo
		return nil
	})
}

func (s *ServicoCombustivelRede) Excluir(idCombustivel, idRede string) error {
	idCombustivel = strings.TrimSpace(idCombustivel)
	idRede = strings.TrimSpace(idRede)
	if idCombustivel == "" || idRede == "" {
		return ErrDadosInvalidos
	}
	return s.repo.Excluir(idCombustivel, idRede)
}

// BuscarPorID retorna o combustível se pertencer à rede (e opcionalmente ao posto).
func (s *ServicoCombustivelRede) BuscarPorID(id, idRede, idPosto string) (*repositorios.CombustivelRedeRegistro, error) {
	id = strings.TrimSpace(id)
	idRede = strings.TrimSpace(idRede)
	idPosto = strings.TrimSpace(idPosto)
	if id == "" || idRede == "" {
		return nil, ErrDadosInvalidos
	}
	reg, err := s.repo.BuscarPorID(id, idRede)
	if err != nil {
		return nil, err
	}
	if idPosto != "" && strings.TrimSpace(reg.PostoID) != idPosto {
		return nil, repositorios.ErrCombustivelRedeNaoEncontrado
	}
	return reg, nil
}
