package servicos

import (
	"fmt"
	"math"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/utils"
)

type ServicoRede interface {
	Listar() ([]*modelos.Rede, error)
	BuscarPorID(id string) (*modelos.Rede, error)
	Criar(input CriarRedeInput) (*modelos.Rede, error)
	Editar(input EditarRedeInput) (*modelos.Rede, error)
	EditarMoedaVirtual(input EditarMoedaVirtualRedeInput) (*modelos.Rede, error)
	EditarVoucherConfig(input EditarVoucherConfigRedeInput) (*modelos.Rede, error)
	EditarGatewayPagamentoModo(idRede, modo string) (*modelos.Rede, error)
	EditarGatewayProvedor(idRede, provedor string, meios modelos.GatewayMeiosHabilitados) (*modelos.Rede, error)
	EditarAppModulos(input EditarAppModulosRedeInput) (*modelos.Rede, error)
	Ativar(id string) (*modelos.Rede, error)
	Desativar(id string) (*modelos.Rede, error)
}

type CriarRedeInput struct {
	NomeFantasia     string
	RazaoSocial      string
	CNPJ             string
	EmailContato     string
	Telefone         string
	ValorImplantacao float64
	ValorMensalidade float64
	PrimeiroCobranca string
}

type EditarRedeInput struct {
	ID               string
	NomeFantasia     string
	RazaoSocial      string
	CNPJ             string
	EmailContato     string
	Telefone         string
	ValorImplantacao float64
	ValorMensalidade float64
	PrimeiroCobranca string
}

type EditarMoedaVirtualRedeInput struct {
	ID                  string
	MoedaVirtualNome    string
	MoedaVirtualCotacao float64
}

// EditarVoucherConfigRedeInput prazos de voucher (app cliente). Campos nulos mantêm o valor atual.
type EditarVoucherConfigRedeInput struct {
	ID      string
	Dias    *int // 1–365, dias para usar o saldo no posto após PIX aprovado
	Minutos *int // 5–10080, janela para concluir o pagamento PIX
}

// EditarAppModulosRedeInput liga/desliga módulos opcionais exibidos no app do cliente.
type EditarAppModulosRedeInput struct {
	ID                     string
	AppModuloIndiqueGanhe  bool
	AppModuloCheckinDiario bool
	AppModuloGireGanhe     bool
	AppModuloRedesSociais  bool
}

type servicoRede struct {
	repo repositorios.RedeRepositorio
}

func NovoServicoRede(repo repositorios.RedeRepositorio) ServicoRede {
	return &servicoRede{repo: repo}
}

func (s *servicoRede) Listar() ([]*modelos.Rede, error) {
	return s.repo.Listar()
}

func (s *servicoRede) BuscarPorID(id string) (*modelos.Rede, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrDadosInvalidos
	}
	return s.repo.BuscarPorID(id)
}

func (s *servicoRede) Criar(input CriarRedeInput) (*modelos.Rede, error) {
	input.NomeFantasia = strings.TrimSpace(input.NomeFantasia)
	input.RazaoSocial = strings.TrimSpace(input.RazaoSocial)
	input.CNPJ = strings.TrimSpace(input.CNPJ)
	input.EmailContato = strings.TrimSpace(input.EmailContato)
	input.Telefone = strings.TrimSpace(input.Telefone)
	input.PrimeiroCobranca = strings.TrimSpace(input.PrimeiroCobranca)

	if input.NomeFantasia == "" || input.RazaoSocial == "" || input.CNPJ == "" || input.PrimeiroCobranca == "" {
		return nil, ErrDadosInvalidos
	}
	if input.ValorImplantacao < 0 || input.ValorMensalidade <= 0 {
		return nil, ErrDadosInvalidos
	}

	primeiraCobranca, err := utils.ParseDataISO(input.PrimeiroCobranca)
	if err != nil {
		return nil, fmt.Errorf("%w: primeiro_cobranca deve estar no formato YYYY-MM-DD", ErrDadosInvalidos)
	}

	rede := &modelos.Rede{
		ID:               utils.GerarToken("rede"),
		NomeFantasia:     input.NomeFantasia,
		RazaoSocial:      input.RazaoSocial,
		CNPJ:             input.CNPJ,
		EmailContato:     input.EmailContato,
		Telefone:         input.Telefone,
		ValorImplantacao: input.ValorImplantacao,
		ValorMensalidade: input.ValorMensalidade,
		PrimeiroCobranca: primeiraCobranca,
		DiaCobranca:      primeiraCobranca.Day(),
		Ativa:            true,
	}
	if err := s.repo.Criar(rede); err != nil {
		return nil, err
	}
	return rede, nil
}

func (s *servicoRede) Editar(input EditarRedeInput) (*modelos.Rede, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.NomeFantasia = strings.TrimSpace(input.NomeFantasia)
	input.RazaoSocial = strings.TrimSpace(input.RazaoSocial)
	input.CNPJ = strings.TrimSpace(input.CNPJ)
	input.EmailContato = strings.TrimSpace(input.EmailContato)
	input.Telefone = strings.TrimSpace(input.Telefone)
	input.PrimeiroCobranca = strings.TrimSpace(input.PrimeiroCobranca)

	if input.ID == "" || input.NomeFantasia == "" || input.RazaoSocial == "" || input.CNPJ == "" || input.PrimeiroCobranca == "" {
		return nil, ErrDadosInvalidos
	}
	if input.ValorImplantacao < 0 || input.ValorMensalidade <= 0 {
		return nil, ErrDadosInvalidos
	}

	primeiraCobranca, err := utils.ParseDataISO(input.PrimeiroCobranca)
	if err != nil {
		return nil, fmt.Errorf("%w: primeiro_cobranca deve estar no formato YYYY-MM-DD", ErrDadosInvalidos)
	}

	return s.repo.Atualizar(input.ID, func(r *modelos.Rede) error {
		r.NomeFantasia = input.NomeFantasia
		r.RazaoSocial = input.RazaoSocial
		r.CNPJ = input.CNPJ
		r.EmailContato = input.EmailContato
		r.Telefone = input.Telefone
		r.ValorImplantacao = input.ValorImplantacao
		r.ValorMensalidade = input.ValorMensalidade
		r.PrimeiroCobranca = primeiraCobranca
		r.DiaCobranca = primeiraCobranca.Day()
		return nil
	})
}

func (s *servicoRede) EditarMoedaVirtual(input EditarMoedaVirtualRedeInput) (*modelos.Rede, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.MoedaVirtualNome = strings.TrimSpace(input.MoedaVirtualNome)
	if input.ID == "" || input.MoedaVirtualNome == "" {
		return nil, ErrDadosInvalidos
	}
	if input.MoedaVirtualCotacao <= 0 || math.IsNaN(input.MoedaVirtualCotacao) || math.IsInf(input.MoedaVirtualCotacao, 0) {
		return nil, ErrDadosInvalidos
	}
	return s.repo.Atualizar(input.ID, func(r *modelos.Rede) error {
		r.MoedaVirtualNome = input.MoedaVirtualNome
		r.MoedaVirtualCotacao = input.MoedaVirtualCotacao
		return nil
	})
}

func (s *servicoRede) EditarGatewayProvedor(idRede, provedor string, meios modelos.GatewayMeiosHabilitados) (*modelos.Rede, error) {
	idRede = strings.TrimSpace(idRede)
	provedor = NormalizarGatewayProvedorAtivo(provedor)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	if err := ValidarMeiosParaProvedor(provedor, meios); err != nil {
		return nil, err
	}
	return s.repo.Atualizar(idRede, func(r *modelos.Rede) error {
		r.GatewayProvedorAtivo = provedor
		r.GatewayMeiosHabilitados = meios
		return nil
	})
}

func (s *servicoRede) EditarGatewayPagamentoModo(idRede, modo string) (*modelos.Rede, error) {
	idRede = strings.TrimSpace(idRede)
	modo = NormalizarGatewayPagamentoModo(modo)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	return s.repo.Atualizar(idRede, func(r *modelos.Rede) error {
		r.GatewayPagamentoModo = modo
		return nil
	})
}

func (s *servicoRede) EditarVoucherConfig(input EditarVoucherConfigRedeInput) (*modelos.Rede, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		return nil, ErrDadosInvalidos
	}
	if input.Dias == nil && input.Minutos == nil {
		return nil, ErrDadosInvalidos
	}
	return s.repo.Atualizar(input.ID, func(r *modelos.Rede) error {
		if input.Dias != nil {
			if *input.Dias < 1 || *input.Dias > 365 {
				return ErrDadosInvalidos
			}
			r.VoucherDiasValidadeResgate = *input.Dias
		}
		if input.Minutos != nil {
			if *input.Minutos < 5 || *input.Minutos > 10080 {
				return ErrDadosInvalidos
			}
			r.VoucherMinutosExpiraPagamentoPix = *input.Minutos
		}
		return nil
	})
}

func (s *servicoRede) EditarAppModulos(input EditarAppModulosRedeInput) (*modelos.Rede, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		return nil, ErrDadosInvalidos
	}
	return s.repo.Atualizar(input.ID, func(r *modelos.Rede) error {
		r.AppModuloIndiqueGanhe = input.AppModuloIndiqueGanhe
		r.AppModuloCheckinDiario = input.AppModuloCheckinDiario
		r.AppModuloGireGanhe = input.AppModuloGireGanhe
		r.AppModuloRedesSociais = input.AppModuloRedesSociais
		return nil
	})
}

func (s *servicoRede) Ativar(id string) (*modelos.Rede, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrDadosInvalidos
	}

	return s.repo.Atualizar(id, func(r *modelos.Rede) error {
		r.Ativa = true
		return nil
	})
}

func (s *servicoRede) Desativar(id string) (*modelos.Rede, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrDadosInvalidos
	}

	return s.repo.Atualizar(id, func(r *modelos.Rede) error {
		r.Ativa = false
		return nil
	})
}
