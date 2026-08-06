package servicos

import (
	"net/url"
	"strings"
	"unicode"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

type ServicoPosto interface {
	ListarPorRedeID(idRede string) ([]*modelos.Posto, error)
	BuscarPorIDNaRede(idPosto, idRede string) (*modelos.Posto, error)
	CriarPostoNaRede(p *modelos.Posto) (*modelos.Posto, error)
	AtualizarPostoNaRede(p *modelos.Posto) (*modelos.Posto, error)
	AtualizarMeiosPosto(idPosto, idRede string, meios modelos.GatewayMeiosHabilitados) (*modelos.Posto, error)
}

type postoRepositorio interface {
	ListarPorRedeID(idRede string) ([]*modelos.Posto, error)
	BuscarPorIDNaRede(idPosto, idRede string) (*modelos.Posto, error)
	Criar(p *modelos.Posto) error
	AtualizarNaRede(p *modelos.Posto) error
	AtualizarMeiosNaRede(idPosto, idRede string, meios modelos.GatewayMeiosHabilitados) (*modelos.Posto, error)
}

type servicoPosto struct {
	repoPosto postoRepositorio
	repoRede  repositorios.RedeRepositorio
}

func NovoServicoPosto(repoPosto postoRepositorio, repoRede repositorios.RedeRepositorio) ServicoPosto {
	return &servicoPosto{
		repoPosto: repoPosto,
		repoRede:  repoRede,
	}
}

func (s *servicoPosto) ListarPorRedeID(idRede string) ([]*modelos.Posto, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.repoRede.BuscarPorID(idRede); err != nil {
		return nil, err
	}
	return s.repoPosto.ListarPorRedeID(idRede)
}

func (s *servicoPosto) BuscarPorIDNaRede(idPosto, idRede string) (*modelos.Posto, error) {
	idPosto = strings.TrimSpace(idPosto)
	idRede = strings.TrimSpace(idRede)
	if idPosto == "" || idRede == "" {
		return nil, ErrDadosInvalidos
	}
	return s.repoPosto.BuscarPorIDNaRede(idPosto, idRede)
}

func apenasDigitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func validarURLLogo(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func normalizarCamposPosto(p *modelos.Posto) {
	if p == nil {
		return
	}
	p.ID = strings.TrimSpace(p.ID)
	p.IDRede = strings.TrimSpace(p.IDRede)
	p.Nome = strings.TrimSpace(p.Nome)
	p.Codigo = strings.TrimSpace(p.Codigo)
	p.NomeFantasia = strings.TrimSpace(p.NomeFantasia)
	p.CNPJ = apenasDigitos(p.CNPJ)
	p.LogoURL = strings.TrimSpace(p.LogoURL)
	p.Rua = strings.TrimSpace(p.Rua)
	p.Numero = strings.TrimSpace(p.Numero)
	p.Bairro = strings.TrimSpace(p.Bairro)
	p.Complemento = strings.TrimSpace(p.Complemento)
	p.CEP = apenasDigitos(p.CEP)
	p.Cidade = strings.TrimSpace(p.Cidade)
	p.Estado = strings.ToUpper(strings.TrimSpace(p.Estado))
	p.Telefone = strings.TrimSpace(p.Telefone)
	p.EmailContato = strings.TrimSpace(p.EmailContato)
}

func validarCamposPostoObrigatorios(p *modelos.Posto) error {
	if p == nil {
		return ErrDadosInvalidos
	}
	if p.IDRede == "" || p.Nome == "" || p.Codigo == "" {
		return ErrDadosInvalidos
	}
	if len(p.CNPJ) != 0 && len(p.CNPJ) != 14 {
		return ErrDadosInvalidos
	}
	if len(p.CEP) != 0 && len(p.CEP) != 8 {
		return ErrDadosInvalidos
	}
	if p.Estado != "" && len(p.Estado) != 2 {
		return ErrDadosInvalidos
	}
	if !validarURLLogo(p.LogoURL) {
		return ErrDadosInvalidos
	}
	return nil
}

func (s *servicoPosto) CriarPostoNaRede(p *modelos.Posto) (*modelos.Posto, error) {
	if p == nil {
		return nil, ErrDadosInvalidos
	}
	normalizarCamposPosto(p)
	if err := validarCamposPostoObrigatorios(p); err != nil {
		return nil, err
	}

	if _, err := s.repoRede.BuscarPorID(p.IDRede); err != nil {
		return nil, err
	}

	if !p.GatewayMeiosHabilitados.TemAlgumMeio() {
		rede, err := s.repoRede.BuscarPorID(p.IDRede)
		if err != nil {
			return nil, err
		}
		p.GatewayMeiosHabilitados = rede.GatewayMeiosHabilitados
	}

	if err := s.repoPosto.Criar(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *servicoPosto) AtualizarPostoNaRede(p *modelos.Posto) (*modelos.Posto, error) {
	if p == nil {
		return nil, ErrDadosInvalidos
	}
	normalizarCamposPosto(p)
	if p.ID == "" {
		return nil, ErrDadosInvalidos
	}
	if err := validarCamposPostoObrigatorios(p); err != nil {
		return nil, err
	}

	if _, err := s.repoRede.BuscarPorID(p.IDRede); err != nil {
		return nil, err
	}

	if err := s.repoPosto.AtualizarNaRede(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *servicoPosto) AtualizarMeiosPosto(idPosto, idRede string, meios modelos.GatewayMeiosHabilitados) (*modelos.Posto, error) {
	idPosto = strings.TrimSpace(idPosto)
	idRede = strings.TrimSpace(idRede)
	if idPosto == "" || idRede == "" {
		return nil, ErrDadosInvalidos
	}
	rede, err := s.repoRede.BuscarPorID(idRede)
	if err != nil {
		return nil, err
	}
	// Posto só pode ligar o que a rede já permite.
	meios = modelos.IntersecaoMeios(rede.GatewayMeiosHabilitados, meios)
	if !meios.TemAlgumMeio() {
		return nil, ErrDadosInvalidos
	}
	if err := ValidarMeiosParaProvedor(NormalizarGatewayProvedorAtivo(rede.GatewayProvedorAtivo), meios); err != nil {
		return nil, err
	}
	return s.repoPosto.AtualizarMeiosNaRede(idPosto, idRede, meios)
}
