package servicos

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"gaspass-servidor/interno/config"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/notificacoes"
	"gaspass-servidor/interno/repositorios"

	"github.com/google/uuid"
)

const (
	prefixoRefVoucherCompra           = "vcompra:"
	tipoRefVoucherCashback            = "voucher_cashback"
	defaultMinutosPagamentoPixVoucher = 30
	defaultDiasValidadeResgateVoucher = 7
	minDiasVoucherResgate             = 1
	maxDiasVoucherResgate             = 365
	minMinutosVoucherPix              = 5
	maxMinutosVoucherPix              = 10080

	intervaloMinConsultaPixProvedor = 20 * time.Second
	intervaloMinConsultaPixForcada  = 10 * time.Second
)

var ultimaConsultaPixProvedor sync.Map // compraID ou compraID:force -> time.Time

// ErrConsultaPixAguarde throttle da verificação manual (botão no app).
var ErrConsultaPixAguarde = errors.New("aguarde alguns segundos antes de consultar de novo")

// VerificarPagamentoPixConsulta resultado da verificação manual no provedor.
type VerificarPagamentoPixConsulta struct {
	Executada bool   `json:"executada"`
	Aprovado  bool   `json:"aprovado"`
	Mensagem  string `json:"mensagem"`
}

// ErrVoucherCampanhaInvalida campanha inexistente ou não aplicável.
var ErrVoucherCampanhaInvalida = errors.New("campanha invalida ou inaplicavel")

// Erros da equipe ao registrar uso (baixa) do voucher no posto.
var (
	ErrVoucherEquipeSemPosto        = errors.New("usuario sem posto vinculado; nao e possivel registrar uso")
	ErrVoucherEquipePapelBaixa      = errors.New("papel nao autorizado a registrar uso do voucher")
	ErrVoucherEquipeNaoAtivoUso     = errors.New("voucher nao esta ativo para uso")
	ErrVoucherEquipeResgateExpirado = errors.New("prazo de resgate do voucher expirou")
	ErrVoucherEquipePostoIncorreto  = errors.New("este voucher so pode ser usado no posto onde foi comprado")
)

// ServicoVoucherCompra compra de voucher no app (PIX + campanha).
type ServicoVoucherCompra struct {
	repo       repositorios.VoucherCompraRepositorio
	campanha   ServicoCampanha
	combustive repositorios.CombustivelRedeRepositorio
	mpGW       repositorios.MercadoPagoGatewayRepositorio
	eredeGW    repositorios.ERedeGatewayRepositorio
	rede       repositorios.RedeRepositorio
	posto      interface {
		BuscarPorIDNaRede(idPosto, idRede string) (*modelos.Posto, error)
	}
	carteira   repositorios.CarteiraRepositorio
	fcm        repositorios.FCMListador
	cfg        config.Config
	indique    *ServicoIndiqueGanhe
	eventos    *ServicoEventosOperacionais
}

func NovoServicoVoucherCompra(
	repo repositorios.VoucherCompraRepositorio,
	camp ServicoCampanha,
	mp repositorios.MercadoPagoGatewayRepositorio,
	erede repositorios.ERedeGatewayRepositorio,
	rede repositorios.RedeRepositorio,
	posto interface {
		BuscarPorIDNaRede(idPosto, idRede string) (*modelos.Posto, error)
	},
	carteira repositorios.CarteiraRepositorio,
	comb repositorios.CombustivelRedeRepositorio,
	fcm repositorios.FCMListador,
	cfg config.Config,
	ind *ServicoIndiqueGanhe,
) *ServicoVoucherCompra {
	return &ServicoVoucherCompra{
		repo: repo, campanha: camp, mpGW: mp, eredeGW: erede, rede: rede, posto: posto,
		carteira: carteira, combustive: comb, fcm: fcm, cfg: cfg, indique: ind,
	}
}

// DefinirEventosOperacionais injeta o serviço de logs/WhatsApp (após construção).
func (s *ServicoVoucherCompra) DefinirEventosOperacionais(ev *ServicoEventosOperacionais) {
	if s != nil {
		s.eventos = ev
	}
}

func (s *ServicoVoucherCompra) registrarEventoVoucher(tipo string, reg *repositorios.VoucherCompraRegistro, extra string) {
	if s == nil || s.eventos == nil || reg == nil {
		return
	}
	idEnt := strings.TrimSpace(reg.ID)
	var entPtr *string
	if idEnt != "" {
		entPtr = &idEnt
	}
	cod := ""
	if reg.CodigoResgate != nil {
		cod = strings.TrimSpace(*reg.CodigoResgate)
	}
	quem := ""
	if s.eventos != nil {
		quem = s.eventos.NomeUsuario(reg.UsuarioID, reg.RedeID)
	}
	titulo := tipo
	switch tipo {
	case modelos.EventoVoucherGerado:
		titulo = "Voucher gerado"
	case modelos.EventoVoucherPago:
		titulo = "Voucher pago"
	case modelos.EventoVoucherBaixa:
		titulo = "Voucher baixa"
	}
	s.eventos.Registrar(RegistrarEventoInput{
		IDRede:       reg.RedeID,
		IDPosto:      reg.PostoCompraID,
		TipoEvento:   tipo,
		EntidadeTipo: "voucher_compra",
		IDEntidade:   entPtr,
		Titulo:       titulo,
		Valor:        FormatValorBR(reg.ValorFinal),
		Quem:         quem,
		Meio:         strings.TrimSpace(reg.MeioPagamento),
		Status:       strings.TrimSpace(reg.Status),
		Codigo:       cod,
		Extra:        extra,
		DataHora:     time.Now().Format("02/01/2006 15:04"),
		Payload: map[string]any{
			"compra_id":      reg.ID,
			"usuario_id":     reg.UsuarioID,
			"valor_final":    reg.ValorFinal,
			"meio_pagamento": reg.MeioPagamento,
			"status":         reg.Status,
		},
	})
}

func (s *ServicoVoucherCompra) duracaoPagamentoPix(idRede string) time.Duration {
	r, err := s.rede.BuscarPorID(idRede)
	if err != nil {
		return defaultMinutosPagamentoPixVoucher * time.Minute
	}
	m := r.VoucherMinutosExpiraPagamentoPix
	if m < minMinutosVoucherPix || m > maxMinutosVoucherPix {
		return defaultMinutosPagamentoPixVoucher * time.Minute
	}
	return time.Duration(m) * time.Minute
}

// expiraResgateAposPagamentoAprovado data/hora limite para uso no posto.
func (s *ServicoVoucherCompra) expiraResgateAposPagamentoAprovado(idRede string, aprovadoEm time.Time) time.Time {
	r, err := s.rede.BuscarPorID(idRede)
	if err != nil {
		return aprovadoEm.Add(defaultDiasValidadeResgateVoucher * 24 * time.Hour)
	}
	d := r.VoucherDiasValidadeResgate
	if d < minDiasVoucherResgate || d > maxDiasVoucherResgate {
		return aprovadoEm.Add(defaultDiasValidadeResgateVoucher * 24 * time.Hour)
	}
	return aprovadoEm.Add(time.Duration(d) * 24 * time.Hour)
}

// ResultadoCalcularVoucher resposta de /v1/eu/vouchers/calcular.
type ResultadoCalcularVoucher struct {
	ValorSolicitado  float64  `json:"valor_solicitado"`
	DescontoAplicado float64  `json:"desconto_aplicado"`
	ValorFinal       float64  `json:"valor_final"`
	TipoBeneficio    string   `json:"tipo_beneficio"`
	CashbackPercentual float64 `json:"cashback_percentual,omitempty"`
	CashbackPrevisto float64  `json:"cashback_previsto,omitempty"`
	Litros           *float64 `json:"litros,omitempty"`
	CampanhaID       *string  `json:"id_campanha,omitempty"`
	CampanhaTitulo   string   `json:"campanha_titulo,omitempty"`
}

// Calcular aplica regras de campanha (sem persistir).
// Para campanha por litro: informe idCombustivelRede e litros; o valor da compra é obtido com preco_por_litro do cadastro.
func (s *ServicoVoucherCompra) Calcular(
	idRede string,
	valor float64,
	idCampanha *string,
	agora time.Time,
	idCombustivelRede *string,
	litros *float64,
) (*ResultadoCalcularVoucher, error) {
	if strings.TrimSpace(idRede) == "" {
		return nil, ErrDadosInvalidos
	}
	if idCampanha == nil || strings.TrimSpace(*idCampanha) == "" {
		// Sem campanha: compra por valor (R$) ou por litro (preço de tabela do combustível).
		if idCombustivelRede != nil && strings.TrimSpace(*idCombustivelRede) != "" && litros != nil && *litros > 1e-9 {
			if s.combustive == nil {
				return nil, ErrDadosInvalidos
			}
			idC := strings.TrimSpace(*idCombustivelRede)
			comb, err := s.combustive.BuscarPorID(idC, idRede)
			if err != nil || !comb.Ativo {
				return nil, ErrVoucherCampanhaInvalida
			}
			valorCompra := round2(comb.PrecoPorLitro * (*litros))
			if valorCompra < 1.0 {
				return nil, ErrDadosInvalidos
			}
			lv := *litros
			return &ResultadoCalcularVoucher{
				ValorSolicitado:  valorCompra,
				ValorFinal:       valorCompra,
				DescontoAplicado: 0,
				Litros:           &lv,
			}, nil
		}
		if valor < 1.0 {
			return nil, ErrDadosInvalidos
		}
		v := round2(valor)
		return &ResultadoCalcularVoucher{ValorSolicitado: v, ValorFinal: v, DescontoAplicado: 0}, nil
	}
	c, err := s.buscarCampanhaElegivel(idRede, strings.TrimSpace(*idCampanha), agora)
	if err != nil {
		return nil, err
	}
	var valorCompra float64
	var litrosVal *float64
	switch c.BaseDesconto {
	case modelos.BaseDescontoLitro:
		if idCombustivelRede == nil || strings.TrimSpace(*idCombustivelRede) == "" || litros == nil || *litros <= 0 {
			return nil, ErrDadosInvalidos
		}
		if c.LitrosMin == nil || c.LitrosMax == nil {
			return nil, ErrDadosInvalidos
		}
		if *litros+1e-9 < *c.LitrosMin || *litros-1e-9 > *c.LitrosMax {
			return nil, ErrDadosInvalidos
		}
		if len(c.IDsCombustiveisRede) == 0 {
			return nil, ErrDadosInvalidos
		}
		idC := strings.TrimSpace(*idCombustivelRede)
		if !slices.Contains(c.IDsCombustiveisRede, idC) {
			return nil, ErrVoucherCampanhaInvalida
		}
		if s.combustive == nil {
			return nil, ErrDadosInvalidos
		}
		comb, err := s.combustive.BuscarPorID(idC, idRede)
		if err != nil || !comb.Ativo {
			return nil, ErrVoucherCampanhaInvalida
		}
		valorCompra = round2(comb.PrecoPorLitro * (*litros))
		if valorCompra < 1.0 {
			return nil, ErrDadosInvalidos
		}
		lv := *litros
		litrosVal = &lv
	case modelos.BaseDescontoValorCompra:
		if valor < 1.0 {
			return nil, ErrDadosInvalidos
		}
		valorCompra = round2(valor)
	default:
		return nil, ErrDadosInvalidos
	}
	if c.BaseDesconto == modelos.BaseDescontoValorCompra {
		if valorCompra+1e-9 < c.ValorMinimoCompra {
			return nil, ErrDadosInvalidos
		}
		if c.ValorMaximoCompra != nil && valorCompra-1e-9 > *c.ValorMaximoCompra {
			return nil, ErrDadosInvalidos
		}
	}
	desconto, err := calcularDescontoCampanha(c, valorCompra, litrosVal)
	if err != nil {
		return nil, err
	}
	beneficio := strings.TrimSpace(c.TipoBeneficio)
	if beneficio == "" {
		beneficio = modelos.TipoBeneficioDesconto
	}
	out := &ResultadoCalcularVoucher{
		ValorSolicitado: valorCompra,
		ValorFinal:      valorCompra,
		DescontoAplicado: 0,
		TipoBeneficio:   beneficio,
	}
	if c.MaxUsosPorCliente != nil {
		// contagem feita em Pagar com usuarioID
	}
	if beneficio == modelos.TipoBeneficioCashback {
		out.CashbackPercentual = normalizarPercentual(c.ValorDesconto)
		out.CashbackPrevisto = floor2(valorCompra * (out.CashbackPercentual / 100.0))
		out.DescontoAplicado = 0
		out.ValorFinal = round2(math.Max(0.01, valorCompra))
	} else {
		out.DescontoAplicado = round2(desconto)
		out.ValorFinal = round2(math.Max(0.01, valorCompra-out.DescontoAplicado))
	}
	out.CampanhaID = idCampanha
	out.CampanhaTitulo = tituloCampanha(c)
	if c.BaseDesconto == modelos.BaseDescontoLitro && litrosVal != nil {
		lr := *litrosVal
		out.Litros = &lr
	}
	return out, nil
}

func (s *ServicoVoucherCompra) buscarCampanhaElegivel(idRede, idCampanha string, agora time.Time) (*modelos.Campanha, error) {
	itens, err := s.campanha.ListarPorRedeID(idRede)
	if err != nil {
		return nil, err
	}
	for _, c := range itens {
		if c != nil && c.ID == idCampanha && repositorios.CampanhaElegivelApp(c, idRede, agora) {
			return c, nil
		}
	}
	return nil, ErrVoucherCampanhaInvalida
}

func tituloCampanha(c *modelos.Campanha) string {
	if t := strings.TrimSpace(c.TituloExibicao); t != "" {
		return t
	}
	if t := strings.TrimSpace(c.Titulo); t != "" {
		return t
	}
	return strings.TrimSpace(c.Nome)
}

func calcularDescontoCampanha(c *modelos.Campanha, valorCompra float64, litros *float64) (float64, error) {
	switch c.ModalidadeDesconto {
	case modelos.ModalidadeDescontoNenhum:
		return 0, nil
	case modelos.ModalidadeDescontoPercentual:
		if c.BaseDesconto == modelos.BaseDescontoLitro {
			// desconto percentual sobre o subtotal (preco*litros)
			if litros == nil {
				return 0, ErrDadosInvalidos
			}
			return valorCompra * (c.ValorDesconto / 100.0), nil
		}
		if c.BaseDesconto != modelos.BaseDescontoValorCompra {
			return 0, ErrDadosInvalidos
		}
		return valorCompra * (c.ValorDesconto / 100.0), nil
	case modelos.ModalidadeDescontoValorFixo:
		if c.BaseDesconto == modelos.BaseDescontoLitro {
			if litros == nil {
				return 0, ErrDadosInvalidos
			}
			d := c.ValorDesconto * (*litros)
			if d > valorCompra-0.01 {
				d = valorCompra - 0.01
			}
			if d < 0 {
				d = 0
			}
			return d, nil
		}
		if c.BaseDesconto != modelos.BaseDescontoValorCompra {
			return 0, ErrDadosInvalidos
		}
		d := c.ValorDesconto
		if d > valorCompra-0.01 {
			d = valorCompra - 0.01
		}
		if d < 0 {
			d = 0
		}
		return d, nil
	default:
		return 0, ErrDadosInvalidos
	}
}

func round2(x float64) float64 {
	return math.Round(x*100) / 100
}

func floor2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Floor(x*100) / 100
}

func floor6(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Floor(x*1_000_000) / 1_000_000
}

func normalizarPercentual(v float64) float64 {
	if v > 0 && v <= 1 {
		return v * 100
	}
	return v
}

// PagarComPixInicia cria cobrança MP e registro local. idPosto obrigatório se a rede usa gateway_pagamento_modo POSTO.
func (s *ServicoVoucherCompra) PagarComPixInicia(ctx context.Context, idRede, idUsuario string, valor float64, idCampanha *string,
	idCombustivelRede *string, litros *float64, idPosto string,
	payerEmail, docTipo, docNumero string, agora time.Time,
) (*repositorios.VoucherCompraRegistro, *PixCobrancaResult, error) {
	if strings.TrimSpace(idRede) == "" || strings.TrimSpace(idUsuario) == "" {
		return nil, nil, ErrDadosInvalidos
	}
	calc, err := s.Calcular(idRede, valor, idCampanha, agora, idCombustivelRede, litros)
	if err != nil {
		return nil, nil, err
	}
	if idCampanha != nil && strings.TrimSpace(*idCampanha) != "" {
		c, err := s.buscarCampanhaElegivel(idRede, strings.TrimSpace(*idCampanha), agora)
		if err != nil {
			return nil, nil, err
		}
		if c.MaxUsosPorCliente != nil {
			n, err := s.repo.ContarUsosCampanhaUsuario(c.ID, idUsuario, idRede)
			if err != nil {
				return nil, nil, err
			}
			if n >= *c.MaxUsosPorCliente {
				return nil, nil, errors.New("limite de usos desta campanha para voce foi atingido")
			}
		}
		pid := strings.TrimSpace(c.IDPosto)
		if pid != "" {
			preq := strings.TrimSpace(idPosto)
			if preq == "" {
				return nil, nil, errors.New("esta campanha e exclusiva de um posto; selecione o posto na compra")
			}
			if pid != preq {
				return nil, nil, errors.New("campanha nao valida para o posto selecionado")
			}
		}
	}
	if calc.ValorFinal < 1.0 {
		return nil, nil, errors.New("valor final apos desconto deve ser pelo menos R$ 1,00")
	}

	rede, err := s.rede.BuscarPorID(idRede)
	if err != nil {
		return nil, nil, err
	}
	if NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo) == modelos.GatewayPagamentoModoPosto &&
		strings.TrimSpace(idPosto) == "" {
		return nil, nil, errors.New("selecione o posto em que vai abastecer")
	}

	gw, err := ResolverGatewayPagamento(s.rede, s.mpGW, s.eredeGW, s.posto, s.cfg, idRede, idPosto)
	if err != nil {
		return nil, nil, err
	}

	idCompra := uuid.New().String()
	ref := prefixoRefVoucherCompra + idCompra
	expP := agora.Add(s.duracaoPagamentoPix(idRede))

	notifURL := gw.MpWebhookURL
	if gw.Provedor == modelos.GatewayProvedorERede {
		notifURL = gw.ERedeWebhookURL
	}
	res, err := CriarPixVoucher(ctx, gw, CriarPixVoucherInput{
		Valor:             calc.ValorFinal,
		Referencia:        ref,
		PayerEmail:        payerEmail,
		DocTipo:           docTipo,
		DocNumero:         docNumero,
		NotificationURL:   notifURL,
	}, expP)
	if err != nil {
		return nil, nil, err
	}
	tid := strings.TrimSpace(res.IDExterno)
	reg := &repositorios.VoucherCompraRegistro{
		ID:                  idCompra,
		RedeID:              idRede,
		UsuarioID:           idUsuario,
		ValorSolicitado:     calc.ValorSolicitado,
		DescontoAplicado:    calc.DescontoAplicado,
		ValorFinal:          calc.ValorFinal,
		TipoBeneficio:       calc.TipoBeneficio,
		CashbackPercentual:  calc.CashbackPercentual,
		CashbackValor:       calc.CashbackPrevisto,
		Status:              "AGUARDANDO_PAGAMENTO",
		MeioPagamento:       modelos.MeioPagamentoPix,
		GatewayProvedor:     gw.Provedor,
		ReferenciaPagamento: &ref,
		ExpiraPagamento:     &expP,
		PostoCompraID:       gw.PostoIDCompra,
	}
	if tid != "" {
		reg.GatewayTID = &tid
	}
	if res.PaymentIDNumerico > 0 {
		mpid := res.PaymentIDNumerico
		reg.MpPaymentID = &mpid
	}
	if idCampanha != nil && strings.TrimSpace(*idCampanha) != "" {
		s := strings.TrimSpace(*idCampanha)
		reg.CampanhaID = &s
	}
	if calc.Litros != nil {
		v := *calc.Litros
		reg.Litros = &v
	}
	if idCombustivelRede != nil && strings.TrimSpace(*idCombustivelRede) != "" {
		s := strings.TrimSpace(*idCombustivelRede)
		reg.CombustivelRedeID = &s
	}
	if err := s.repo.CriarPendenteComPix(reg); err != nil {
		return nil, res, err
	}
	logPixVoucherCriado(idRede, reg, gw, res)
	s.registrarEventoVoucher(modelos.EventoVoucherGerado, reg, "")
	return reg, res, nil
}

// PagarComDinheiroInicia cria voucher com código imediato (pagamento ao frentista no posto).
func (s *ServicoVoucherCompra) PagarComDinheiroInicia(idRede, idUsuario string, valor float64, idCampanha *string,
	idCombustivelRede *string, litros *float64, idPosto string, agora time.Time,
) (*repositorios.VoucherCompraRegistro, error) {
	if strings.TrimSpace(idRede) == "" || strings.TrimSpace(idUsuario) == "" {
		return nil, ErrDadosInvalidos
	}
	calc, err := s.Calcular(idRede, valor, idCampanha, agora, idCombustivelRede, litros)
	if err != nil {
		return nil, err
	}
	if idCampanha != nil && strings.TrimSpace(*idCampanha) != "" {
		c, err := s.buscarCampanhaElegivel(idRede, strings.TrimSpace(*idCampanha), agora)
		if err != nil {
			return nil, err
		}
		if c.MaxUsosPorCliente != nil {
			n, err := s.repo.ContarUsosCampanhaUsuario(c.ID, idUsuario, idRede)
			if err != nil {
				return nil, err
			}
			if n >= *c.MaxUsosPorCliente {
				return nil, errors.New("limite de usos desta campanha para voce foi atingido")
			}
		}
		pid := strings.TrimSpace(c.IDPosto)
		if pid != "" {
			preq := strings.TrimSpace(idPosto)
			if preq == "" {
				return nil, errors.New("esta campanha e exclusiva de um posto; selecione o posto na compra")
			}
			if pid != preq {
				return nil, errors.New("campanha nao valida para o posto selecionado")
			}
		}
	}
	if calc.ValorFinal < 1.0 {
		return nil, errors.New("valor final apos desconto deve ser pelo menos R$ 1,00")
	}

	rede, err := s.rede.BuscarPorID(idRede)
	if err != nil {
		return nil, err
	}
	meios := rede.GatewayMeiosHabilitados
	preq := strings.TrimSpace(idPosto)
	if preq != "" && s.posto != nil {
		posto, errP := s.posto.BuscarPorIDNaRede(preq, idRede)
		if errP != nil {
			return nil, errP
		}
		meios = modelos.IntersecaoMeios(rede.GatewayMeiosHabilitados, posto.GatewayMeiosHabilitados)
	}
	if !meios.Dinheiro {
		if preq != "" {
			return nil, errors.New("este posto nao aceita pagamento em dinheiro no momento")
		}
		return nil, errors.New("rede nao aceita pagamento em dinheiro no momento")
	}
	modo := NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo)
	var postoCompra *string
	if modo == modelos.GatewayPagamentoModoPosto {
		p := strings.TrimSpace(idPosto)
		if p == "" {
			return nil, errors.New("selecione o posto em que vai abastecer")
		}
		postoCompra = &p
	} else if strings.TrimSpace(idPosto) != "" {
		p := strings.TrimSpace(idPosto)
		postoCompra = &p
	}

	idCompra := uuid.New().String()
	cod := gerarCodigoResgate()
	expR := s.expiraResgateAposPagamentoAprovado(idRede, agora)
	reg := &repositorios.VoucherCompraRegistro{
		ID:                 idCompra,
		RedeID:             idRede,
		UsuarioID:          idUsuario,
		ValorSolicitado:    calc.ValorSolicitado,
		DescontoAplicado:   calc.DescontoAplicado,
		ValorFinal:         calc.ValorFinal,
		TipoBeneficio:      calc.TipoBeneficio,
		CashbackPercentual: calc.CashbackPercentual,
		CashbackValor:      calc.CashbackPrevisto,
		Status:             "AGUARDANDO_DINHEIRO",
		MeioPagamento:      modelos.MeioPagamentoDinheiro,
		CodigoResgate:      &cod,
		ExpiraResgate:      &expR,
		PostoCompraID:      postoCompra,
	}
	if idCampanha != nil && strings.TrimSpace(*idCampanha) != "" {
		s := strings.TrimSpace(*idCampanha)
		reg.CampanhaID = &s
	}
	if calc.Litros != nil {
		v := *calc.Litros
		reg.Litros = &v
	}
	if idCombustivelRede != nil && strings.TrimSpace(*idCombustivelRede) != "" {
		s := strings.TrimSpace(*idCombustivelRede)
		reg.CombustivelRedeID = &s
	}
	var lastErr error
	for range 8 {
		lastErr = s.repo.CriarAguardandoDinheiro(reg)
		if lastErr == nil {
			log.Printf("voucher_dinheiro criado: rede=%s compra=%s codigo=%s", idRede, reg.ID, cod)
			s.registrarEventoVoucher(modelos.EventoVoucherGerado, reg, "")
			return reg, nil
		}
		if !strings.Contains(strings.ToLower(lastErr.Error()), "unique") &&
			!strings.Contains(strings.ToLower(lastErr.Error()), "duplicate") {
			return nil, lastErr
		}
		cod = gerarCodigoResgate()
		reg.CodigoResgate = &cod
	}
	return nil, fmt.Errorf("falha ao gerar codigo unico: %w", lastErr)
}

func (s *ServicoVoucherCompra) enriquecerConsultaEquipe(out *repositorios.VoucherCompraConsultaEquipe, operadorPostoID string) {
	if out == nil {
		return
	}
	if strings.TrimSpace(out.MeioPagamento) == "" {
		out.MeioPagamento = modelos.MeioPagamentoPix
	}
	aguarda := out.Status == "AGUARDANDO_DINHEIRO"
	if aguarda {
		out.MeioPagamento = modelos.MeioPagamentoDinheiro
	}
	out.AguardaPagamentoDinheiro = aguarda
	if aguarda {
		restante := ValorRestanteACobrar(&out.VoucherCompraRegistro)
		if UsouMoedaVirtual(&out.VoucherCompraRegistro) && restante > 0 {
			out.AvisoTitulo = "Cobrar restante em dinheiro"
			out.AvisoCorpo = fmt.Sprintf(
				"Cliente ja usou moeda virtual. Receba apenas R$ %.2f em dinheiro e confirme abaixo.",
				restante,
			)
		} else {
			out.AvisoTitulo = "Cobrar pagamento em dinheiro"
			out.AvisoCorpo = "Este voucher ainda nao foi pago. Receba o valor em dinheiro do cliente e confirme abaixo para liberar o abastecimento."
		}
	}

	compraPosto := ""
	if out.PostoCompraID != nil {
		compraPosto = strings.TrimSpace(*out.PostoCompraID)
	}
	if compraPosto == "" {
		return
	}
	out.UsoRestritoAoPostoCompra = true
	if s.posto != nil {
		if p, err := s.posto.BuscarPorIDNaRede(compraPosto, out.RedeID); err == nil && p != nil {
			nome := strings.TrimSpace(p.NomeFantasia)
			if nome == "" {
				nome = strings.TrimSpace(p.Nome)
			}
			out.PostoCompraNome = nome
		}
	}
	op := strings.TrimSpace(operadorPostoID)
	if op == "" {
		return
	}
	ok := strings.EqualFold(op, compraPosto)
	out.OperadorPodeRegistrarUso = &ok
	if ok {
		return
	}
	if out.PostoCompraNome != "" {
		out.OperadorAvisoPosto = fmt.Sprintf(
			"Este voucher so pode ser usado em %s. Voce esta em outro posto.",
			out.PostoCompraNome,
		)
	} else {
		out.OperadorAvisoPosto = ErrVoucherEquipePostoIncorreto.Error()
	}
}

// ConsultarPorCodigoResgateEquipe voucher por código de resgate na rede (frentista / gerente / gestor).
// operadorPostoID: posto do token (frentista/gerente); vazio para gestor sem posto.
func (s *ServicoVoucherCompra) ConsultarPorCodigoResgateEquipe(idRede, codigo, operadorPostoID string) (*repositorios.VoucherCompraConsultaEquipe, error) {
	idRede = strings.TrimSpace(idRede)
	codigo = strings.TrimSpace(codigo)
	if idRede == "" || codigo == "" {
		return nil, ErrDadosInvalidos
	}
	out, err := s.repo.BuscarPorCodigoResgateConsultaEquipe(codigo, idRede)
	if err != nil {
		return nil, err
	}
	s.enriquecerConsultaEquipe(out, operadorPostoID)
	return out, nil
}

func logPixVoucherCriado(idRede string, reg *repositorios.VoucherCompraRegistro, gw *GatewayContext, pix *PixCobrancaResult) {
	if reg == nil || gw == nil || pix == nil {
		return
	}
	posto := ""
	if reg.PostoCompraID != nil {
		posto = strings.TrimSpace(*reg.PostoCompraID)
	}
	mpID := int64(0)
	if reg.MpPaymentID != nil {
		mpID = *reg.MpPaymentID
	}
	tid := ""
	if reg.GatewayTID != nil {
		tid = strings.TrimSpace(*reg.GatewayTID)
	}
	qrLen := len(strings.TrimSpace(pix.QrCode))
	log.Printf(
		"voucher_pix criado: rede=%s compra=%s provedor=%s modo_gateway=%s posto_compra=%s "+
			"gateway_payment_id=%s mp_payment_id=%d tid=%s erede_ambiente=%s qr_len=%d mp_status=%s",
		strings.TrimSpace(idRede),
		reg.ID,
		strings.TrimSpace(reg.GatewayProvedor),
		gw.Modo,
		posto,
		strings.TrimSpace(pix.IDExterno),
		mpID,
		tid,
		strings.TrimSpace(gw.ERedeAmbiente),
		qrLen,
		strings.TrimSpace(pix.Status),
	)
}

// RetomarDadosPixPendente reconsulta o payment no MP e devolve o QR (na DB só há mp_payment_id, não a string do QR).
// Útil para o cliente reabrir o ecrã PIX a partir da lista "aguardando pagamento".
func (s *ServicoVoucherCompra) RetomarDadosPixPendente(ctx context.Context, idCompra, idRede, idUsuario string) (
	reg *repositorios.VoucherCompraRegistro, pix *PixCobrancaResult, err error,
) {
	vc, err := s.repo.BuscarPorID(idCompra, idUsuario, idRede)
	if err != nil {
		return nil, nil, err
	}
	if vc.Status != "AGUARDANDO_PAGAMENTO" {
		return nil, nil, errors.New("este voucher nao esta a aguardar pagamento")
	}
	if vc.MpPaymentID == nil && (vc.GatewayTID == nil || strings.TrimSpace(*vc.GatewayTID) == "") {
		return nil, nil, errors.New("pagamento nao associado a esta compra")
	}
	if vc.ExpiraPagamento != nil && time.Now().After(*vc.ExpiraPagamento) {
		return nil, nil, errors.New("prazo de pagamento deste pix expirou; gere outro voucher")
	}
	idPosto := ""
	if vc.PostoCompraID != nil {
		idPosto = strings.TrimSpace(*vc.PostoCompraID)
	}
	gw, err := ResolverGatewayPagamento(s.rede, s.mpGW, s.eredeGW, s.posto, s.cfg, idRede, idPosto)
	if err != nil {
		return nil, nil, err
	}
	provedor := strings.TrimSpace(vc.GatewayProvedor)
	if provedor == "" {
		provedor = gw.Provedor
	}
	pix, err = ConsultarPixVoucher(ctx, gw, provedor, gatewayTIDStr(vc), vc.MpPaymentID)
	if err != nil {
		return nil, nil, err
	}
	switch strings.TrimSpace(pix.Status) {
	case "approved":
		return nil, nil, errors.New("pagamento ja confirmado; actualize a lista de vouchers")
	case "rejected", "cancelled", "refunded", "charged_back":
		return nil, nil, fmt.Errorf("cobranca nao esta pendente (status: %s)", pix.Status)
	}
	if strings.TrimSpace(pix.QrCode) == "" {
		return nil, nil, errors.New("qr pix indisponivel; tente gerar outro pagamento no app")
	}
	log.Printf(
		"voucher_pix retomar: rede=%s compra=%s provedor=%s modo_gateway=%s posto_compra=%s gateway_payment_id=%s mp_status=%s",
		strings.TrimSpace(idRede),
		vc.ID,
		provedor,
		gw.Modo,
		idPosto,
		strings.TrimSpace(pix.IDExterno),
		strings.TrimSpace(pix.Status),
	)
	return vc, pix, nil
}

func gatewayTIDStr(vc *repositorios.VoucherCompraRegistro) string {
	if vc == nil || vc.GatewayTID == nil {
		return ""
	}
	return strings.TrimSpace(*vc.GatewayTID)
}

// ListarMeus do cliente.
func (s *ServicoVoucherCompra) ListarMeus(rede, usuarioID string) ([]*repositorios.VoucherCompraRegistro, error) {
	return s.repo.ListarDoUsuario(rede, usuarioID, 80)
}

// UsosAprovadosPorCampanha contagem (pagamento aprovado: ATIVO ou USADO) por campanha, para 1/x no app.
func (s *ServicoVoucherCompra) UsosAprovadosPorCampanha(rede, usuarioID string) (map[string]int, error) {
	if strings.TrimSpace(rede) == "" || strings.TrimSpace(usuarioID) == "" {
		return nil, ErrDadosInvalidos
	}
	return s.repo.ListarUsosAprovadosPorCampanha(rede, usuarioID)
}

// BuscarMeu de um registro. Se PIX pendente, consulta o provedor (fallback quando webhook não chegou).
func (s *ServicoVoucherCompra) BuscarMeu(id, rede, usuario string) (*repositorios.VoucherCompraRegistro, error) {
	vc, err := s.repo.BuscarPorID(id, usuario, rede)
	if err != nil {
		return nil, err
	}
	if vc.Status == "AGUARDANDO_PAGAMENTO" {
		s.tentarSincronizarStatusPixPendente(context.Background(), vc, rede, false)
		return s.repo.BuscarPorID(id, usuario, rede)
	}
	return vc, nil
}

// VerificarPagamentoPixCliente consulta o provedor na hora (botão "Já paguei") e devolve status atualizado.
func (s *ServicoVoucherCompra) VerificarPagamentoPixCliente(ctx context.Context, id, rede, usuario string) (
	*repositorios.VoucherCompraRegistro, *VerificarPagamentoPixConsulta, error,
) {
	id = strings.TrimSpace(id)
	rede = strings.TrimSpace(rede)
	usuario = strings.TrimSpace(usuario)
	if id == "" || rede == "" || usuario == "" {
		return nil, nil, ErrDadosInvalidos
	}
	if !podeConsultarPixForcadaAgora(id) {
		return nil, nil, ErrConsultaPixAguarde
	}
	marcarConsultaPixForcada(id)

	vc, err := s.repo.BuscarPorID(id, usuario, rede)
	if err != nil {
		return nil, nil, err
	}
	consulta := &VerificarPagamentoPixConsulta{}

	switch strings.TrimSpace(vc.Status) {
	case "ATIVO":
		consulta.Executada = false
		consulta.Aprovado = true
		consulta.Mensagem = "Seu voucher já está ativo."
		return vc, consulta, nil
	case "USADO":
		consulta.Aprovado = true
		consulta.Mensagem = "Este voucher já foi utilizado."
		return vc, consulta, nil
	case "CANCELADO":
		consulta.Mensagem = "Este pagamento foi cancelado."
		return vc, consulta, nil
	case "EXPIRADO":
		consulta.Mensagem = "O prazo deste pagamento PIX expirou."
		return vc, consulta, nil
	case "AGUARDANDO_PAGAMENTO":
		if vc.ExpiraPagamento != nil && time.Now().After(*vc.ExpiraPagamento) {
			consulta.Mensagem = "O prazo deste pagamento PIX expirou."
			return vc, consulta, nil
		}
		s.tentarSincronizarStatusPixPendente(ctx, vc, rede, true)
		consulta.Executada = true
		vc2, err := s.repo.BuscarPorID(id, usuario, rede)
		if err != nil {
			return nil, consulta, err
		}
		if strings.TrimSpace(vc2.Status) == "ATIVO" {
			consulta.Aprovado = true
			consulta.Mensagem = "Pagamento confirmado! Seu voucher está pronto."
		} else {
			consulta.Mensagem = "Pagamento ainda não confirmado pelo banco. Tente novamente em alguns instantes."
		}
		return vc2, consulta, nil
	default:
		consulta.Mensagem = "Status do pagamento indisponível."
		return vc, consulta, nil
	}
}

func podeConsultarPixForcadaAgora(compraID string) bool {
	compraID = strings.TrimSpace(compraID)
	if compraID == "" {
		return true
	}
	key := compraID + ":force"
	now := time.Now()
	if v, ok := ultimaConsultaPixProvedor.Load(key); ok {
		if t, ok := v.(time.Time); ok && now.Sub(t) < intervaloMinConsultaPixForcada {
			return false
		}
	}
	return true
}

func marcarConsultaPixForcada(compraID string) {
	compraID = strings.TrimSpace(compraID)
	if compraID == "" {
		return
	}
	ultimaConsultaPixProvedor.Store(compraID+":force", time.Now())
}

func deveConsultarPixProvedorAgora(compraID string) bool {
	compraID = strings.TrimSpace(compraID)
	if compraID == "" {
		return true
	}
	now := time.Now()
	if v, ok := ultimaConsultaPixProvedor.Load(compraID); ok {
		if t, ok := v.(time.Time); ok && now.Sub(t) < intervaloMinConsultaPixProvedor {
			return false
		}
	}
	ultimaConsultaPixProvedor.Store(compraID, now)
	return true
}

func logPixSyncPendenteThrottled(compraID, msg string) {
	compraID = strings.TrimSpace(compraID)
	if compraID == "" {
		log.Print(msg)
		return
	}
	key := compraID + ":pendente"
	now := time.Now()
	if v, ok := ultimaConsultaPixProvedor.Load(key); ok {
		if t, ok := v.(time.Time); ok && now.Sub(t) < 60*time.Second {
			return
		}
	}
	ultimaConsultaPixProvedor.Store(key, now)
	log.Print(msg)
}

// tentarSincronizarStatusPixPendente consulta e.Rede/MP e ativa o voucher se o pagamento já foi aprovado.
// forcarConsulta ignora o throttle de 20s (verificação manual no app).
func (s *ServicoVoucherCompra) tentarSincronizarStatusPixPendente(ctx context.Context, vc *repositorios.VoucherCompraRegistro, idRede string, forcarConsulta bool) {
	if vc == nil || vc.Status != "AGUARDANDO_PAGAMENTO" {
		return
	}
	if vc.ExpiraPagamento != nil && time.Now().After(*vc.ExpiraPagamento) {
		log.Printf("voucher_pix sync: expirado compra=%s", vc.ID)
		return
	}
	tid := gatewayTIDStr(vc)
	if vc.MpPaymentID == nil && tid == "" {
		log.Printf("voucher_pix sync: sem tid/mp compra=%s", vc.ID)
		return
	}
	if !forcarConsulta {
		if !deveConsultarPixProvedorAgora(vc.ID) {
			return
		}
	} else {
		ultimaConsultaPixProvedor.Store(strings.TrimSpace(vc.ID), time.Now())
	}
	idPosto := ""
	if vc.PostoCompraID != nil {
		idPosto = strings.TrimSpace(*vc.PostoCompraID)
	}
	gw, err := ResolverGatewayPagamento(s.rede, s.mpGW, s.eredeGW, s.posto, s.cfg, idRede, idPosto)
	if err != nil {
		log.Printf("voucher_pix sync: gateway compra=%s posto=%s: %v", vc.ID, idPosto, err)
		return
	}
	provedor := strings.TrimSpace(vc.GatewayProvedor)
	if provedor == "" {
		provedor = gw.Provedor
	}
	log.Printf(
		"voucher_pix sync: consultando compra=%s provedor=%s tid=%s posto=%s",
		vc.ID, provedor, tid, idPosto,
	)
	pix, err := ConsultarPixVoucher(ctx, gw, provedor, tid, vc.MpPaymentID)
	if err != nil {
		log.Printf("voucher_pix sync: consulta compra=%s provedor=%s: %v", vc.ID, provedor, err)
		return
	}
	if strings.TrimSpace(pix.Status) != "approved" {
		logPixSyncPendenteThrottled(vc.ID, fmt.Sprintf(
			"voucher_pix sync: aguardando compra=%s provedor=%s tid=%s status_provedor=%q",
			vc.ID, provedor, tid, strings.TrimSpace(pix.GatewayStatusLabel),
		))
		return
	}
	log.Printf(
		"voucher_pix sync: aprovado no provedor compra=%s provedor=%s id_externo=%s — ativando voucher",
		vc.ID, provedor, strings.TrimSpace(pix.IDExterno),
	)
	if vc.ReferenciaPagamento != nil && strings.TrimSpace(*vc.ReferenciaPagamento) != "" {
		s.ProcessarPagamentoAprovadoPorReferencia(idRede, *vc.ReferenciaPagamento)
		return
	}
	s.processarAtivacaoVoucher(idRede, vc.ID)
}

// RegistrarBaixaPorCodigoEquipe marca o voucher ATIVO ou AGUARDANDO_DINHEIRO como USADO.
func (s *ServicoVoucherCompra) RegistrarBaixaPorCodigoEquipe(u *modelos.UsuarioSessao, codigo string, idPostoOpcional *string) (*repositorios.VoucherCompraConsultaEquipe, error) {
	if u == nil {
		return nil, ErrDadosInvalidos
	}
	codigo = strings.TrimSpace(codigo)
	if strings.TrimSpace(u.IDRede) == "" || strings.TrimSpace(u.IDUsuario) == "" || codigo == "" {
		return nil, ErrDadosInvalidos
	}
	var postoPtr *string
	switch u.Papel {
	case modelos.PapelFrentista, modelos.PapelGerentePosto:
		p := strings.TrimSpace(u.IDPosto)
		if p == "" {
			return nil, ErrVoucherEquipeSemPosto
		}
		postoPtr = &p
	case modelos.PapelGestorRede, modelos.PapelSuperAdmin:
		if idPostoOpcional != nil {
			p := strings.TrimSpace(*idPostoOpcional)
			if p != "" {
				postoPtr = &p
			}
		}
	default:
		return nil, ErrVoucherEquipePapelBaixa
	}
	vc, err := s.repo.BuscarPorCodigoResgateConsultaEquipe(codigo, u.IDRede)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(vc.Status)
	// AGUARDANDO_DINHEIRO já implica pagamento em dinheiro no posto (código gerado na criação).
	ehDinheiro := status == "AGUARDANDO_DINHEIRO"
	ehAtivo := status == "ATIVO"
	if !ehDinheiro && !ehAtivo {
		return nil, ErrVoucherEquipeNaoAtivoUso
	}
	if vc.ExpiraResgate != nil && time.Now().After(*vc.ExpiraResgate) {
		return nil, ErrVoucherEquipeResgateExpirado
	}
	if vc.PostoCompraID != nil && strings.TrimSpace(*vc.PostoCompraID) != "" {
		compraPosto := strings.TrimSpace(*vc.PostoCompraID)
		operPosto := ""
		if postoPtr != nil {
			operPosto = strings.TrimSpace(*postoPtr)
		}
		if !strings.EqualFold(operPosto, compraPosto) {
			log.Printf(
				"voucher baixa: posto incorreto codigo=%s compra_posto=%s operador_posto=%s papel=%s",
				codigo, compraPosto, operPosto, u.Papel,
			)
			return nil, ErrVoucherEquipePostoIncorreto
		}
	}
	if err := s.repo.RegistrarBaixaUso(vc.ID, u.IDRede, postoPtr, u.IDUsuario, string(u.Papel), strings.TrimSpace(u.NomeCompleto)); err != nil {
		log.Printf("voucher baixa: RegistrarBaixaUso id=%s: %v", vc.ID, err)
		return nil, err
	}
	uid := strings.TrimSpace(vc.UsuarioID)
	cod := ""
	if vc.CodigoResgate != nil {
		cod = strings.TrimSpace(*vc.CodigoResgate)
	}
	if ehDinheiro {
		if s.indique != nil && uid != "" && !UsouMoedaVirtual(&vc.VoucherCompraRegistro) {
			s.indique.AposVoucherAprovado(u.IDRede, uid, vc.ID)
		}
		s.creditarCashbackVoucher(u.IDRede, &vc.VoucherCompraRegistro)
	}
	go s.notificarPushVoucherUsadoNoPosto(uid, vc.ID, cod, vc.ValorFinal, ehDinheiro)
	out, err := s.repo.BuscarPorCodigoResgateConsultaEquipe(codigo, u.IDRede)
	if err != nil {
		return nil, err
	}
	opPosto := ""
	if postoPtr != nil {
		opPosto = strings.TrimSpace(*postoPtr)
	}
	s.enriquecerConsultaEquipe(out, opPosto)
	regEv := out.VoucherCompraRegistro
	if (regEv.PostoCompraID == nil || strings.TrimSpace(*regEv.PostoCompraID) == "") && postoPtr != nil {
		regEv.PostoCompraID = postoPtr
	}
	s.registrarEventoVoucher(modelos.EventoVoucherBaixa, &regEv, strings.TrimSpace(u.NomeCompleto))
	return out, nil
}

// ListarPainelPorRede compras da rede para o painel (gestor, equipe, super-admin); status vazio = todos.
func (s *ServicoVoucherCompra) ListarPainelPorRede(idRede string, limite, offset int, status string) ([]*repositorios.VoucherCompraPainelLinha, int, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, 0, ErrDadosInvalidos
	}
	if limite < 1 {
		limite = 50
	}
	if limite > 200 {
		limite = 200
	}
	if offset < 0 {
		offset = 0
	}
	status = strings.TrimSpace(status)
	if status != "" {
		switch status {
		case "AGUARDANDO_PAGAMENTO", "AGUARDANDO_DINHEIRO", "ATIVO", "USADO", "EXPIRADO", "CANCELADO":
		default:
			return nil, 0, ErrDadosInvalidos
		}
	}
	return s.repo.ListarPainelPorRede(idRede, limite, offset, status)
}

// ProcessarPagamentoAprovadoPorReferencia ativa voucher após pagamento confirmado (MP ou e.Rede).
func (s *ServicoVoucherCompra) ProcessarPagamentoAprovadoPorReferencia(idRede, ref string) {
	ref = strings.TrimSpace(ref)
	idCompra, ok := parseRefVcompra(ref)
	if !ok {
		return
	}
	s.processarAtivacaoVoucher(idRede, idCompra)
}

// ProcessarPagamentoAprovadoMercadoPago webhook MP (external_reference = vcompra:uuid).
func (s *ServicoVoucherCompra) ProcessarPagamentoAprovadoMercadoPago(idRede, ref string) {
	s.ProcessarPagamentoAprovadoPorReferencia(idRede, ref)
}

// ProcessarPagamentoAprovadoERede webhook e.Rede por tid (busca referencia na compra).
func (s *ServicoVoucherCompra) ProcessarPagamentoAprovadoERede(idRede, tid string) {
	tid = strings.TrimSpace(tid)
	if tid == "" {
		return
	}
	// Buscar por gateway_tid via BuscarPorIDRede após achar compra - usar repo se existir
	// Fallback: consultar não temos list by tid - add repo method or search
	vc, err := s.repo.BuscarPorGatewayTIDRede(tid, idRede)
	if err != nil {
		log.Printf("voucher erede webhook: buscar tid=%s: %v", tid, err)
		return
	}
	if vc.ReferenciaPagamento != nil {
		s.ProcessarPagamentoAprovadoPorReferencia(idRede, *vc.ReferenciaPagamento)
		return
	}
	s.processarAtivacaoVoucher(idRede, vc.ID)
}

func (s *ServicoVoucherCompra) processarAtivacaoVoucher(idRede, idCompra string) {
	vc, err := s.repo.BuscarPorIDRede(idCompra, idRede)
	if err != nil {
		log.Printf("voucher webhook: buscar %s: %v", idCompra, err)
		return
	}
	if vc.Status == "ATIVO" {
		s.creditarCashbackVoucher(idRede, vc)
		return
	}
	if vc.Status != "AGUARDANDO_PAGAMENTO" {
		return
	}
	cod := gerarCodigoResgate()
	var lastErr error
	for range 8 {
		lastErr = s.repo.AtivarPagamentoAprovado(idCompra, idRede, cod, s.expiraResgateAposPagamentoAprovado(idRede, time.Now()))
		if lastErr == nil {
			log.Printf("voucher webhook: ativado id=%s codigo=%s", idCompra, cod)
			uid := strings.TrimSpace(vc.UsuarioID)
			if s.indique != nil && uid != "" && !UsouMoedaVirtual(vc) {
				s.indique.AposVoucherAprovado(idRede, uid, idCompra)
			}
			s.creditarCashbackVoucher(idRede, vc)
			go s.notificarPushVoucherAprovado(uid, idCompra, cod, vc.ValorFinal)
			vc.Status = "ATIVO"
			vc.CodigoResgate = &cod
			s.registrarEventoVoucher(modelos.EventoVoucherPago, vc, "")
			return
		}
		if strings.Contains(lastErr.Error(), "nenhuma linha ativada") {
			return
		}
		cod = gerarCodigoResgate()
	}
	log.Printf("voucher webhook: falha ativar id=%s: %v", idCompra, lastErr)
}

func (s *ServicoVoucherCompra) creditarCashbackVoucher(idRede string, vc *repositorios.VoucherCompraRegistro) {
	if vc == nil || strings.TrimSpace(vc.TipoBeneficio) != modelos.TipoBeneficioCashback {
		return
	}
	if vc.CashbackValor <= 0 || vc.CashbackCreditadoEm != nil {
		return
	}
	if s.carteira == nil {
		log.Printf("voucher cashback: carteira indisponivel compra=%s", strings.TrimSpace(vc.ID))
		return
	}
	uid := strings.TrimSpace(vc.UsuarioID)
	if uid == "" {
		return
	}
	rede, err := s.rede.BuscarPorID(idRede)
	if err != nil {
		log.Printf("voucher cashback: buscar rede %s: %v", idRede, err)
		return
	}
	cotacao := rede.MoedaVirtualCotacao
	if cotacao <= 0 {
		log.Printf("voucher cashback: cotacao invalida rede=%s", idRede)
		return
	}
	carteiraID, err := s.carteira.ObterOuCriarCarteira(idRede, uid, strings.TrimSpace(rede.MoedaVirtualNome), cotacao)
	if err != nil {
		log.Printf("voucher cashback: obter carteira usuario=%s: %v", uid, err)
		return
	}
	valorFiat := floor2(vc.CashbackValor)
	valorToken := floor6(valorFiat / cotacao)
	if valorFiat <= 0 || valorToken <= 0 {
		return
	}
	if err := s.carteira.CreditarCashback(idRede, carteiraID, valorFiat, valorToken, tipoRefVoucherCashback, vc.ID); err != nil {
		log.Printf("voucher cashback: creditar compra=%s: %v", vc.ID, err)
		return
	}
	if ok, err := s.repo.MarcarCashbackCreditado(vc.ID, idRede, time.Now()); err != nil {
		log.Printf("voucher cashback: marcar creditado compra=%s: %v", vc.ID, err)
	} else if ok {
		log.Printf("voucher cashback: creditado compra=%s valor=%0.2f", vc.ID, valorFiat)
	}
}

func (s *ServicoVoucherCompra) notificarPushVoucherAprovado(idUsuario, idCompra, codigo string, valor float64) {
	if s.fcm == nil {
		return
	}
	cred := strings.TrimSpace(s.cfg.FcmCaminhoContaServico)
	if cred == "" {
		return
	}
	if strings.TrimSpace(idUsuario) == "" {
		return
	}
	tokens, err := s.fcm.ListarTokensFCMPorUsuarioID(idUsuario)
	if err != nil {
		log.Printf("fcm: listar tokens usuario=%s: %v", idUsuario, err)
		return
	}
	if len(tokens) == 0 {
		return
	}
	v := formatarBRL2(valor)
	xctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	notificacoes.EnviarVoucherAprovado(xctx, cred, tokens, idCompra, codigo, v)
}

func (s *ServicoVoucherCompra) notificarPushVoucherUsadoNoPosto(idUsuario, idCompra, codigo string, valor float64, dinheiro bool) {
	if s.fcm == nil {
		return
	}
	cred := strings.TrimSpace(s.cfg.FcmCaminhoContaServico)
	if cred == "" {
		return
	}
	if strings.TrimSpace(idUsuario) == "" {
		return
	}
	tokens, err := s.fcm.ListarTokensFCMPorUsuarioID(idUsuario)
	if err != nil {
		log.Printf("fcm: listar tokens usuario=%s: %v", idUsuario, err)
		return
	}
	if len(tokens) == 0 {
		return
	}
	v := formatarBRL2(valor)
	xctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	notificacoes.EnviarVoucherUsadoNoPosto(xctx, cred, tokens, idCompra, codigo, v, dinheiro)
}

func formatarBRL2(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return s[:i] + "," + s[i+1:]
	}
	return s
}

func parseRefVcompra(ref string) (string, bool) {
	if !strings.HasPrefix(ref, prefixoRefVoucherCompra) {
		return "", false
	}
	id := strings.TrimSpace(ref[len(prefixoRefVoucherCompra):])
	if id == "" {
		return "", false
	}
	return id, true
}

func gerarCodigoResgate() string {
	const alfabeto = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	s := make([]byte, 8)
	for i := range s {
		s[i] = alfabeto[int(buf[i])%len(alfabeto)]
	}
	return string(s)
}
