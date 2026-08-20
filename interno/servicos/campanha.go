package servicos

import (
	"math"
	"net/url"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

type ServicoCampanha interface {
	ListarPorRedeID(idRede string) ([]*modelos.Campanha, error)
	BuscarPorIDeRede(idCampanha, idRede string) (*modelos.Campanha, error)
	Criar(sessaoCriador string, in CriarCampanhaInput) (*modelos.Campanha, error)
	Atualizar(in AtualizarCampanhaInput) error
}

type CriarCampanhaInput struct {
	IDRede                string
	Nome                  string
	Titulo                string
	Descricao             string
	ImagemURL             string
	IDPosto               string
	VigenciaInicio        *time.Time
	VigenciaFim           *time.Time
	Status                modelos.StatusCampanha
	ValidaNoApp           bool
	ValidaNoPostoFisico   bool
	TipoBeneficio         string
	ModalidadeDesconto    string
	BaseDesconto          string
	ValorDesconto         float64
	ValorMinimoCompra     float64
	ValorMaximoCompra     *float64
	MaxUsosPorCliente     *int
	LitrosMin             *float64
	LitrosMax             *float64
	IDsCombustiveisRede   []string
}

type AtualizarCampanhaInput struct {
	ID                  string
	IDRede              string
	Nome                string
	Titulo              string
	Descricao           string
	ImagemURL           string
	IDPosto             string
	VigenciaInicio      *time.Time
	VigenciaFim         *time.Time
	Status              modelos.StatusCampanha
	ValidaNoApp         bool
	ValidaNoPostoFisico bool
	TipoBeneficio       string
	ModalidadeDesconto  string
	BaseDesconto        string
	ValorDesconto         float64
	ValorMinimoCompra     float64
	ValorMaximoCompra     *float64
	MaxUsosPorCliente     *int
	LitrosMin             *float64
	LitrosMax             *float64
	IDsCombustiveisRede   []string
}

type campanhaRepo interface {
	ListarPorRedeID(idRede string) ([]*modelos.Campanha, error)
	BuscarPorIDeRede(idCampanha, idRede string) (*modelos.Campanha, error)
	Criar(sessaoCriador string, c *modelos.Campanha) error
	Atualizar(c *modelos.Campanha) error
	PostoPertenceARede(idPosto, idRede string) (bool, error)
}

type servicoCampanha struct {
	repo     campanhaRepo
	repoRede repositorios.RedeRepositorio
	repoComb repositorios.CombustivelRedeRepositorio
}

func NovoServicoCampanha(repo campanhaRepo, repoRede repositorios.RedeRepositorio, repoComb repositorios.CombustivelRedeRepositorio) ServicoCampanha {
	return &servicoCampanha{repo: repo, repoRede: repoRede, repoComb: repoComb}
}

func (s *servicoCampanha) BuscarPorIDeRede(idCampanha, idRede string) (*modelos.Campanha, error) {
	c, err := s.repo.BuscarPorIDeRede(idCampanha, idRede)
	if err != nil {
		return nil, err
	}
	aplicarLegadoCampanhaCashback(c)
	return c, nil
}

func validarURLImagem(s string) bool {
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

func statusCampanhaValido(s modelos.StatusCampanha) bool {
	switch s {
	case modelos.StatusCampanhaRascunho, modelos.StatusCampanhaAtiva,
		modelos.StatusCampanhaPausada, modelos.StatusCampanhaArquivada:
		return true
	default:
		return false
	}
}

func normalizarModalidade(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	switch s {
	case modelos.ModalidadeDescontoNenhum:
		return modelos.ModalidadeDescontoNenhum
	case modelos.ModalidadeDescontoPercentual, modelos.ModalidadeDescontoValorFixo:
		return s
	default:
		return modelos.ModalidadeDescontoNenhum
	}
}

func normalizarBase(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == modelos.BaseDescontoLitro {
		return modelos.BaseDescontoLitro
	}
	return modelos.BaseDescontoValorCompra
}

func normalizarTipoBeneficio(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == modelos.TipoBeneficioCashback {
		return modelos.TipoBeneficioCashback
	}
	return modelos.TipoBeneficioDesconto
}

// validarFaixaValorCompra exige piso e teto em R$ para base VALOR_COMPRA (teto >= piso).
func validarFaixaValorCompra(vmin float64, vmax *float64) bool {
	if math.IsNaN(vmin) || math.IsInf(vmin, 0) || vmin < 0 {
		return false
	}
	if vmax == nil || math.IsNaN(*vmax) || math.IsInf(*vmax, 0) {
		return false
	}
	return *vmax >= vmin
}

func validarDescontoEUso(
	validaApp, validaPosto bool,
	tipoBeneficio, mod, base string,
	valorDesc, vmin float64,
	maxUsos *int,
) bool {
	// Exatamente um canal: app OU posto fisico (mutuamente exclusivos).
	if validaApp == validaPosto {
		return false
	}
	if math.IsNaN(valorDesc) || math.IsInf(valorDesc, 0) {
		return false
	}
	if math.IsNaN(vmin) || math.IsInf(vmin, 0) || vmin < 0 {
		return false
	}
	if maxUsos != nil && *maxUsos < 1 {
		return false
	}
	switch mod {
	case modelos.ModalidadeDescontoNenhum:
		if valorDesc != 0 && math.Abs(valorDesc) > 1e-9 {
			return false
		}
	case modelos.ModalidadeDescontoPercentual:
		if valorDesc <= 0 || valorDesc > 100 {
			return false
		}
	case modelos.ModalidadeDescontoValorFixo:
		if valorDesc <= 0 {
			return false
		}
	default:
		return false
	}
	if tipoBeneficio == modelos.TipoBeneficioCashback {
		if mod != modelos.ModalidadeDescontoPercentual {
			return false
		}
	}
	switch base {
	case modelos.BaseDescontoValorCompra, modelos.BaseDescontoLitro:
		return true
	default:
		return false
	}
}

func (s *servicoCampanha) ListarPorRedeID(idRede string) ([]*modelos.Campanha, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	if _, err := s.repoRede.BuscarPorID(idRede); err != nil {
		return nil, err
	}
	itens, err := s.repo.ListarPorRedeID(idRede)
	if err != nil {
		return nil, err
	}
	for _, c := range itens {
		aplicarLegadoCampanhaCashback(c)
	}
	return itens, nil
}

func baseDescontoSolicitadaBruta(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func dedupIDsCombustiveis(ids []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *servicoCampanha) validarCombustiveisAtivosNaRede(idRede, idPosto string, ids []string) error {
	if len(ids) == 0 {
		return ErrDadosInvalidos
	}
	todos, err := s.repoComb.ListarPorRede(idRede, "")
	if err != nil {
		return err
	}
	ativos := make(map[string]*repositorios.CombustivelRedeRegistro, len(todos))
	for _, c := range todos {
		if c != nil && c.Ativo {
			ativos[c.ID] = c
		}
	}
	idPosto = strings.TrimSpace(idPosto)
	for _, id := range ids {
		c, ok := ativos[id]
		if !ok {
			return ErrDadosInvalidos
		}
		if idPosto != "" && strings.TrimSpace(c.PostoID) != idPosto {
			return ErrDadosInvalidos
		}
	}
	return nil
}

// expandirCombustiveisCampanha: escopo rede expande por código/nome em todos os postos;
// escopo posto mantém só os IDs (já validados do posto).
func (s *servicoCampanha) expandirCombustiveisCampanha(idRede, idPosto string, ids []string) ([]string, error) {
	ids = dedupIDsCombustiveis(ids)
	if len(ids) == 0 {
		return nil, ErrDadosInvalidos
	}
	idPosto = strings.TrimSpace(idPosto)
	if idPosto != "" {
		if err := s.validarCombustiveisAtivosNaRede(idRede, idPosto, ids); err != nil {
			return nil, err
		}
		return ids, nil
	}
	if err := s.validarCombustiveisAtivosNaRede(idRede, "", ids); err != nil {
		return nil, err
	}
	exp, err := s.repoComb.ExpandirIDsMesmoTipo(idRede, ids)
	if err != nil {
		return nil, err
	}
	if len(exp) == 0 {
		return nil, ErrDadosInvalidos
	}
	return dedupIDsCombustiveis(exp), nil
}

func (s *servicoCampanha) Criar(sessaoCriador string, in CriarCampanhaInput) (*modelos.Campanha, error) {
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.Nome = strings.TrimSpace(in.Nome)
	in.Titulo = strings.TrimSpace(in.Titulo)
	in.Descricao = strings.TrimSpace(in.Descricao)
	in.ImagemURL = strings.TrimSpace(in.ImagemURL)
	in.IDPosto = strings.TrimSpace(in.IDPosto)
	in.ValidaNoApp, in.ValidaNoPostoFisico = true, false
	if baseDescontoSolicitadaBruta(in.BaseDesconto) == modelos.BaseDescontoUnidade {
		return nil, ErrDadosInvalidos
	}
	mod := normalizarModalidade(in.ModalidadeDesconto)
	tipoBeneficio := normalizarTipoBeneficio(in.TipoBeneficio)
	base := normalizarBase(in.BaseDesconto)
	valor := normalizarPercentualArmazenamento(mod, in.ValorDesconto)
	if mod == modelos.ModalidadeDescontoNenhum {
		valor = 0
	}
	var litMinPtr, litMaxPtr *float64
	idsComb := dedupIDsCombustiveis(in.IDsCombustiveisRede)
	if base == modelos.BaseDescontoLitro {
		if mod == modelos.ModalidadeDescontoNenhum {
			return nil, ErrDadosInvalidos
		}
		if in.LitrosMin == nil || in.LitrosMax == nil {
			return nil, ErrDadosInvalidos
		}
		if *in.LitrosMin <= 0 || *in.LitrosMax < *in.LitrosMin {
			return nil, ErrDadosInvalidos
		}
		litMinPtr = in.LitrosMin
		litMaxPtr = in.LitrosMax
	} else {
		idsComb = nil
	}

	if in.IDRede == "" || in.Nome == "" {
		return nil, ErrDadosInvalidos
	}
	if in.VigenciaInicio == nil || in.VigenciaFim == nil {
		return nil, ErrDadosInvalidos
	}
	if in.VigenciaFim.Before(*in.VigenciaInicio) {
		return nil, ErrDadosInvalidos
	}
	if !validarURLImagem(in.ImagemURL) {
		return nil, ErrDadosInvalidos
	}
	if in.Status == "" {
		in.Status = modelos.StatusCampanhaAtiva
	}
	if !statusCampanhaValido(in.Status) {
		return nil, ErrDadosInvalidos
	}
	vminUse := in.ValorMinimoCompra
	var vmaxUse *float64
	if base == modelos.BaseDescontoLitro {
		vminUse = 0
		vmaxUse = nil
	} else {
		vmaxUse = in.ValorMaximoCompra
		if !validarFaixaValorCompra(vminUse, vmaxUse) {
			return nil, ErrDadosInvalidos
		}
	}
	if !validarDescontoEUso(in.ValidaNoApp, in.ValidaNoPostoFisico, tipoBeneficio, mod, base, valor, vminUse, in.MaxUsosPorCliente) {
		return nil, ErrDadosInvalidos
	}

	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return nil, err
	}

	if in.IDPosto != "" {
		ok, err := s.repo.PostoPertenceARede(in.IDPosto, in.IDRede)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, repositorios.ErrPostoNaoPertenceARede
		}
	}

	if base == modelos.BaseDescontoLitro {
		exp, err := s.expandirCombustiveisCampanha(in.IDRede, in.IDPosto, idsComb)
		if err != nil {
			return nil, err
		}
		idsComb = exp
	}

	c := &modelos.Campanha{
		IDRede:               in.IDRede,
		Nome:                 in.Nome,
		Titulo:               in.Titulo,
		Descricao:            in.Descricao,
		ImagemURL:            in.ImagemURL,
		IDPosto:              in.IDPosto,
		Status:               in.Status,
		VigenciaInicio:       in.VigenciaInicio,
		VigenciaFim:          in.VigenciaFim,
		ValidaNoApp:          in.ValidaNoApp,
		ValidaNoPostoFisico:  in.ValidaNoPostoFisico,
		TipoBeneficio:        tipoBeneficio,
		ModalidadeDesconto:   mod,
		BaseDesconto:         base,
		ValorDesconto:        valor,
		ValorMinimoCompra:    vminUse,
		ValorMaximoCompra:    vmaxUse,
		MaxUsosPorCliente:    in.MaxUsosPorCliente,
		LitrosMin:            litMinPtr,
		LitrosMax:            litMaxPtr,
		IDsCombustiveisRede:  idsComb,
	}
	if err := s.repo.Criar(sessaoCriador, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *servicoCampanha) Atualizar(in AtualizarCampanhaInput) error {
	in.ID = strings.TrimSpace(in.ID)
	in.IDRede = strings.TrimSpace(in.IDRede)
	in.Nome = strings.TrimSpace(in.Nome)
	in.Titulo = strings.TrimSpace(in.Titulo)
	in.Descricao = strings.TrimSpace(in.Descricao)
	in.ImagemURL = strings.TrimSpace(in.ImagemURL)
	in.IDPosto = strings.TrimSpace(in.IDPosto)
	in.ValidaNoApp, in.ValidaNoPostoFisico = true, false
	if baseDescontoSolicitadaBruta(in.BaseDesconto) == modelos.BaseDescontoUnidade {
		return ErrDadosInvalidos
	}
	mod := normalizarModalidade(in.ModalidadeDesconto)
	tipoBeneficio := normalizarTipoBeneficio(in.TipoBeneficio)
	base := normalizarBase(in.BaseDesconto)
	valor := normalizarPercentualArmazenamento(mod, in.ValorDesconto)
	if mod == modelos.ModalidadeDescontoNenhum {
		valor = 0
	}
	var litMinPtr, litMaxPtr *float64
	idsComb := dedupIDsCombustiveis(in.IDsCombustiveisRede)
	if base == modelos.BaseDescontoLitro {
		if mod == modelos.ModalidadeDescontoNenhum {
			return ErrDadosInvalidos
		}
		if in.LitrosMin == nil || in.LitrosMax == nil {
			return ErrDadosInvalidos
		}
		if *in.LitrosMin <= 0 || *in.LitrosMax < *in.LitrosMin {
			return ErrDadosInvalidos
		}
		litMinPtr = in.LitrosMin
		litMaxPtr = in.LitrosMax
	} else {
		idsComb = nil
	}

	if in.ID == "" || in.IDRede == "" || in.Nome == "" {
		return ErrDadosInvalidos
	}
	if in.VigenciaInicio == nil || in.VigenciaFim == nil {
		return ErrDadosInvalidos
	}
	if in.VigenciaFim.Before(*in.VigenciaInicio) {
		return ErrDadosInvalidos
	}
	if in.Status == "" {
		return ErrDadosInvalidos
	}
	if !validarURLImagem(in.ImagemURL) {
		return ErrDadosInvalidos
	}
	if !statusCampanhaValido(in.Status) {
		return ErrDadosInvalidos
	}
	vminUse := in.ValorMinimoCompra
	var vmaxUse *float64
	if base == modelos.BaseDescontoLitro {
		vminUse = 0
		vmaxUse = nil
	} else {
		vmaxUse = in.ValorMaximoCompra
		if !validarFaixaValorCompra(vminUse, vmaxUse) {
			return ErrDadosInvalidos
		}
	}
	if !validarDescontoEUso(in.ValidaNoApp, in.ValidaNoPostoFisico, tipoBeneficio, mod, base, valor, vminUse, in.MaxUsosPorCliente) {
		return ErrDadosInvalidos
	}

	if _, err := s.repoRede.BuscarPorID(in.IDRede); err != nil {
		return err
	}

	if in.IDPosto != "" {
		ok, err := s.repo.PostoPertenceARede(in.IDPosto, in.IDRede)
		if err != nil {
			return err
		}
		if !ok {
			return repositorios.ErrPostoNaoPertenceARede
		}
	}

	if base == modelos.BaseDescontoLitro {
		exp, err := s.expandirCombustiveisCampanha(in.IDRede, in.IDPosto, idsComb)
		if err != nil {
			return err
		}
		idsComb = exp
	}

	c := &modelos.Campanha{
		ID:                  in.ID,
		IDRede:              in.IDRede,
		Nome:                in.Nome,
		Titulo:              in.Titulo,
		Descricao:           in.Descricao,
		ImagemURL:           in.ImagemURL,
		IDPosto:             in.IDPosto,
		Status:              in.Status,
		VigenciaInicio:      in.VigenciaInicio,
		VigenciaFim:         in.VigenciaFim,
		ValidaNoApp:         in.ValidaNoApp,
		ValidaNoPostoFisico: in.ValidaNoPostoFisico,
		TipoBeneficio:       tipoBeneficio,
		ModalidadeDesconto:  mod,
		BaseDesconto:        base,
		ValorDesconto:       valor,
		ValorMinimoCompra:   vminUse,
		ValorMaximoCompra:   vmaxUse,
		MaxUsosPorCliente:   in.MaxUsosPorCliente,
		LitrosMin:           litMinPtr,
		LitrosMax:           litMaxPtr,
		IDsCombustiveisRede: idsComb,
	}
	return s.repo.Atualizar(c)
}
