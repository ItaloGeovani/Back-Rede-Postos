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
)

// ErrVoucherCampanhaInvalida campanha inexistente ou não aplicável.
var ErrVoucherCampanhaInvalida = errors.New("campanha invalida ou inaplicavel")

// Erros da equipe ao registrar uso (baixa) do voucher no posto.
var (
	ErrVoucherEquipeSemPosto        = errors.New("usuario sem posto vinculado; nao e possivel registrar uso")
	ErrVoucherEquipePapelBaixa      = errors.New("papel nao autorizado a registrar uso do voucher")
	ErrVoucherEquipeNaoAtivoUso     = errors.New("voucher nao esta ativo para uso")
	ErrVoucherEquipeResgateExpirado = errors.New("prazo de resgate do voucher expirou")
)

// ServicoVoucherCompra compra de voucher no app (PIX + campanha).
type ServicoVoucherCompra struct {
	repo       repositorios.VoucherCompraRepositorio
	campanha   ServicoCampanha
	combustive repositorios.CombustivelRedeRepositorio
	mpGW       repositorios.MercadoPagoGatewayRepositorio
	eredeGW    repositorios.ERedeGatewayRepositorio
	rede       repositorios.RedeRepositorio
	carteira   repositorios.CarteiraRepositorio
	fcm        repositorios.FCMListador
	cfg        config.Config
	indique    *ServicoIndiqueGanhe
}

func NovoServicoVoucherCompra(
	repo repositorios.VoucherCompraRepositorio,
	camp ServicoCampanha,
	mp repositorios.MercadoPagoGatewayRepositorio,
	erede repositorios.ERedeGatewayRepositorio,
	rede repositorios.RedeRepositorio,
	carteira repositorios.CarteiraRepositorio,
	comb repositorios.CombustivelRedeRepositorio,
	fcm repositorios.FCMListador,
	cfg config.Config,
	ind *ServicoIndiqueGanhe,
) *ServicoVoucherCompra {
	return &ServicoVoucherCompra{repo: repo, campanha: camp, mpGW: mp, eredeGW: erede, rede: rede, carteira: carteira, combustive: comb, fcm: fcm, cfg: cfg, indique: ind}
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

	gw, err := ResolverGatewayPagamento(s.rede, s.mpGW, s.eredeGW, s.cfg, idRede, idPosto)
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
	return reg, res, nil
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
	gw, err := ResolverGatewayPagamento(s.rede, s.mpGW, s.eredeGW, s.cfg, idRede, idPosto)
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
		s.tentarSincronizarStatusPixPendente(context.Background(), vc, rede)
		return s.repo.BuscarPorID(id, usuario, rede)
	}
	return vc, nil
}

// tentarSincronizarStatusPixPendente consulta e.Rede/MP e ativa o voucher se o pagamento já foi aprovado.
func (s *ServicoVoucherCompra) tentarSincronizarStatusPixPendente(ctx context.Context, vc *repositorios.VoucherCompraRegistro, idRede string) {
	if vc == nil || vc.Status != "AGUARDANDO_PAGAMENTO" {
		return
	}
	if vc.ExpiraPagamento != nil && time.Now().After(*vc.ExpiraPagamento) {
		return
	}
	if vc.MpPaymentID == nil && gatewayTIDStr(vc) == "" {
		return
	}
	idPosto := ""
	if vc.PostoCompraID != nil {
		idPosto = strings.TrimSpace(*vc.PostoCompraID)
	}
	gw, err := ResolverGatewayPagamento(s.rede, s.mpGW, s.eredeGW, s.cfg, idRede, idPosto)
	if err != nil {
		return
	}
	provedor := strings.TrimSpace(vc.GatewayProvedor)
	if provedor == "" {
		provedor = gw.Provedor
	}
	pix, err := ConsultarPixVoucher(ctx, gw, provedor, gatewayTIDStr(vc), vc.MpPaymentID)
	if err != nil {
		log.Printf("voucher_pix sync: consulta compra=%s provedor=%s: %v", vc.ID, provedor, err)
		return
	}
	if strings.TrimSpace(pix.Status) != "approved" {
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

// ConsultarPorCodigoResgateEquipe voucher por código de resgate na rede (frentista / gerente / gestor).
func (s *ServicoVoucherCompra) ConsultarPorCodigoResgateEquipe(idRede, codigo string) (*repositorios.VoucherCompraConsultaEquipe, error) {
	idRede = strings.TrimSpace(idRede)
	codigo = strings.TrimSpace(codigo)
	if idRede == "" || codigo == "" {
		return nil, ErrDadosInvalidos
	}
	return s.repo.BuscarPorCodigoResgateConsultaEquipe(codigo, idRede)
}

// RegistrarBaixaPorCodigoEquipe marca o voucher ATIVO como USADO e grava posto + operador (frentista/gerente/gestor).
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
	if strings.TrimSpace(vc.Status) != "ATIVO" {
		return nil, ErrVoucherEquipeNaoAtivoUso
	}
	if vc.ExpiraResgate != nil && time.Now().After(*vc.ExpiraResgate) {
		return nil, ErrVoucherEquipeResgateExpirado
	}
	if vc.PostoCompraID != nil && strings.TrimSpace(*vc.PostoCompraID) != "" {
		compraPosto := strings.TrimSpace(*vc.PostoCompraID)
		if postoPtr == nil || strings.TrimSpace(*postoPtr) != compraPosto {
			return nil, errors.New("este voucher so pode ser usado no posto onde foi comprado")
		}
	}
	if err := s.repo.RegistrarBaixaUso(vc.ID, u.IDRede, postoPtr, u.IDUsuario, string(u.Papel), strings.TrimSpace(u.NomeCompleto)); err != nil {
		return nil, err
	}
	return s.repo.BuscarPorCodigoResgateConsultaEquipe(codigo, u.IDRede)
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
		case "AGUARDANDO_PAGAMENTO", "ATIVO", "USADO", "EXPIRADO", "CANCELADO":
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
			if s.indique != nil && uid != "" {
				s.indique.AposVoucherAprovado(idRede, uid, idCompra)
			}
			s.creditarCashbackVoucher(idRede, vc)
			go s.notificarPushVoucherAprovado(uid, idCompra, cod, vc.ValorFinal)
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
