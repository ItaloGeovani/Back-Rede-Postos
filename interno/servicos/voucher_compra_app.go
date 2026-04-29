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
	"github.com/mercadopago/sdk-go/pkg/payment"
)

const (
	prefixoRefVoucherCompra           = "vcompra:"
	defaultMinutosPagamentoPixVoucher = 30
	defaultDiasValidadeResgateVoucher = 7
	minDiasVoucherResgate             = 1
	maxDiasVoucherResgate             = 365
	minMinutosVoucherPix              = 5
	maxMinutosVoucherPix              = 10080
)

// ErrVoucherCampanhaInvalida campanha inexistente ou não aplicável.
var ErrVoucherCampanhaInvalida = errors.New("campanha invalida ou inaplicavel")

// ServicoVoucherCompra compra de voucher no app (PIX + campanha).
type ServicoVoucherCompra struct {
	repo       repositorios.VoucherCompraRepositorio
	campanha   ServicoCampanha
	combustive repositorios.CombustivelRedeRepositorio
	mpGW       repositorios.MercadoPagoGatewayRepositorio
	rede       repositorios.RedeRepositorio
	fcm        repositorios.FCMListador
	cfg        config.Config
	indique    *ServicoIndiqueGanhe
}

func NovoServicoVoucherCompra(
	repo repositorios.VoucherCompraRepositorio,
	camp ServicoCampanha,
	mp repositorios.MercadoPagoGatewayRepositorio,
	rede repositorios.RedeRepositorio,
	comb repositorios.CombustivelRedeRepositorio,
	fcm repositorios.FCMListador,
	cfg config.Config,
	ind *ServicoIndiqueGanhe,
) *ServicoVoucherCompra {
	return &ServicoVoucherCompra{repo: repo, campanha: camp, mpGW: mp, rede: rede, combustive: comb, fcm: fcm, cfg: cfg, indique: ind}
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
	out := &ResultadoCalcularVoucher{ValorSolicitado: valorCompra, ValorFinal: valorCompra, DescontoAplicado: 0}
	if c.MaxUsosPorCliente != nil {
		// contagem feita em Pagar com usuarioID
	}
	out.DescontoAplicado = round2(desconto)
	out.ValorFinal = round2(math.Max(0.01, valorCompra-out.DescontoAplicado))
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

// PagarComPixInicia cria cobrança MP e registro local.
func (s *ServicoVoucherCompra) PagarComPixInicia(ctx context.Context, idRede, idUsuario string, valor float64, idCampanha *string,
	idCombustivelRede *string, litros *float64,
	payerEmail, docTipo, docNumero string, agora time.Time,
) (*repositorios.VoucherCompraRegistro, *payment.Response, error) {
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
	}
	if calc.ValorFinal < 1.0 {
		return nil, nil, errors.New("valor final apos desconto deve ser pelo menos R$ 1,00")
	}

	creds, err := s.mpGW.BuscarPorRedeID(idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrMercadoPagoGatewayNaoConfigurado) {
			return nil, nil, errors.New("rede sem mercado pago configurado")
		}
		return nil, nil, err
	}
	if strings.TrimSpace(creds.AccessToken) == "" {
		return nil, nil, errors.New("rede sem mp_access_token")
	}
	base := strings.TrimRight(strings.TrimSpace(s.cfg.PublicBaseURL), "/")
	if base == "" {
		return nil, nil, errors.New("servidor sem PUBLIC_BASE_URL")
	}
	notif := base + "/v1/public/mercadopago/webhook/" + idRede

	idCompra := uuid.New().String()
	ref := prefixoRefVoucherCompra + idCompra
	expP := agora.Add(s.duracaoPagamentoPix(idRede))

	res, err := CriarCobrancaPixMercadoPago(ctx, creds.AccessToken, CriarCobrancaPixMercadoPagoInput{
		Valor:             calc.ValorFinal,
		Descricao:         "Voucher Auto Posto",
		PayerEmail:        payerEmail,
		DocTipo:           docTipo,
		DocNumero:         docNumero,
		ExternalReference: ref,
		NotificationURL:   notif,
	})
	if err != nil {
		return nil, nil, err
	}
	mpid := int64(res.ID)
	reg := &repositorios.VoucherCompraRegistro{
		ID:                  idCompra,
		RedeID:              idRede,
		UsuarioID:           idUsuario,
		ValorSolicitado:     calc.ValorSolicitado,
		DescontoAplicado:    calc.DescontoAplicado,
		ValorFinal:          calc.ValorFinal,
		Status:              "AGUARDANDO_PAGAMENTO",
		MpPaymentID:         &mpid,
		ReferenciaPagamento: &ref,
		ExpiraPagamento:     &expP,
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
	return reg, res, nil
}

// RetomarDadosPixPendente reconsulta o payment no MP e devolve o QR (na DB só há mp_payment_id, não a string do QR).
// Útil para o cliente reabrir o ecrã PIX a partir da lista "aguardando pagamento".
func (s *ServicoVoucherCompra) RetomarDadosPixPendente(ctx context.Context, idCompra, idRede, idUsuario string) (
	reg *repositorios.VoucherCompraRegistro, pay *payment.Response, err error,
) {
	vc, err := s.repo.BuscarPorID(idCompra, idUsuario, idRede)
	if err != nil {
		return nil, nil, err
	}
	if vc.Status != "AGUARDANDO_PAGAMENTO" {
		return nil, nil, errors.New("este voucher nao esta a aguardar pagamento")
	}
	if vc.MpPaymentID == nil {
		return nil, nil, errors.New("pagamento mercado pago nao associado a esta compra")
	}
	if vc.ExpiraPagamento != nil && time.Now().After(*vc.ExpiraPagamento) {
		return nil, nil, errors.New("prazo de pagamento deste pix expirou; gere outro voucher")
	}
	creds, err := s.mpGW.BuscarPorRedeID(idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrMercadoPagoGatewayNaoConfigurado) {
			return nil, nil, errors.New("rede sem mercado pago configurado")
		}
		return nil, nil, err
	}
	if strings.TrimSpace(creds.AccessToken) == "" {
		return nil, nil, errors.New("rede sem mp_access_token")
	}
	pay, err = ConsultarPagamentoMercadoPago(ctx, creds.AccessToken, int(*vc.MpPaymentID))
	if err != nil {
		return nil, nil, err
	}
	switch strings.TrimSpace(pay.Status) {
	case "approved":
		return nil, nil, errors.New("pagamento ja confirmado; actualize a lista de vouchers")
	case "rejected", "cancelled", "refunded", "charged_back":
		return nil, nil, fmt.Errorf("cobranca nao esta pendente (status: %s)", pay.Status)
	}
	qr, _ := ExtrairQRPixDoPagamento(pay)
	if strings.TrimSpace(qr) == "" {
		return nil, nil, errors.New("qr pix indisponivel; tente gerar outro pagamento no app")
	}
	return vc, pay, nil
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

// BuscarMeu de um registro.
func (s *ServicoVoucherCompra) BuscarMeu(id, rede, usuario string) (*repositorios.VoucherCompraRegistro, error) {
	return s.repo.BuscarPorID(id, usuario, rede)
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

// ProcessarPagamentoAprovadoWebhook chamado do webhook MP quando o pagamento está approved.
func (s *ServicoVoucherCompra) ProcessarPagamentoAprovadoWebhook(idRede string, pay *payment.Response) {
	if pay == nil {
		return
	}
	ref := strings.TrimSpace(pay.ExternalReference)
	idCompra, ok := parseRefVcompra(ref)
	if !ok {
		return
	}
	vc, err := s.repo.BuscarPorIDRede(idCompra, idRede)
	if err != nil {
		log.Printf("voucher webhook: buscar %s: %v", idCompra, err)
		return
	}
	if vc.Status == "ATIVO" {
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
