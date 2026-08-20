package servicos

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"

	"github.com/google/uuid"
)

const tipoRefVoucherMoeda = "voucher_moeda"

// PagarComMoedaInicia compra com débito imediato de moeda virtual (total ou parcial).
// Campanha não é permitida. Restante 0 → ATIVO; restante > 0 → PIX ou DINHEIRO só do restante.
// Débito é irreversível (sem estorno se o restante expirar).
func (s *ServicoVoucherCompra) PagarComMoedaInicia(
	ctx context.Context,
	idRede, idUsuario string,
	valor float64,
	idCombustivelRede *string,
	litros *float64,
	idPosto string,
	valorMoedaFiat float64,
	meioRestante string,
	payerEmail, docTipo, docNumero string,
	agora time.Time,
) (*repositorios.VoucherCompraRegistro, *PixCobrancaResult, error) {
	if strings.TrimSpace(idRede) == "" || strings.TrimSpace(idUsuario) == "" {
		return nil, nil, ErrDadosInvalidos
	}
	if s.carteira == nil {
		return nil, nil, errors.New("carteira indisponivel")
	}
	valorMoedaFiat = round2(valorMoedaFiat)
	if valorMoedaFiat <= 0 {
		return nil, nil, errors.New("informe valor_moeda_fiat maior que zero")
	}

	calc, err := s.Calcular(idRede, valor, nil, agora, idCombustivelRede, litros, idPosto)
	if err != nil {
		return nil, nil, err
	}
	if calc.ValorFinal < 1.0 {
		return nil, nil, errors.New("valor final deve ser pelo menos R$ 1,00")
	}
	if valorMoedaFiat > calc.ValorFinal+1e-9 {
		return nil, nil, errors.New("valor em moeda nao pode exceder o valor do voucher")
	}

	rede, err := s.rede.BuscarPorID(idRede)
	if err != nil {
		return nil, nil, err
	}
	meios := rede.GatewayMeiosHabilitados
	preq := strings.TrimSpace(idPosto)
	if preq != "" && s.posto != nil {
		posto, errP := s.posto.BuscarPorIDNaRede(preq, idRede)
		if errP != nil {
			return nil, nil, errP
		}
		meios = modelos.IntersecaoMeios(rede.GatewayMeiosHabilitados, posto.GatewayMeiosHabilitados)
	}
	if !meios.MoedaVirtual {
		if preq != "" {
			return nil, nil, errors.New("este posto nao aceita pagamento com moeda virtual no momento")
		}
		return nil, nil, errors.New("rede nao aceita pagamento com moeda virtual no momento")
	}

	cotacao := rede.MoedaVirtualCotacao
	if cotacao <= 0 {
		return nil, nil, errors.New("cotacao da moeda virtual invalida")
	}
	valorToken := floor6(valorMoedaFiat / cotacao)
	if valorToken <= 0 {
		return nil, nil, errors.New("valor em moeda invalido apos cotacao")
	}
	if _, err := s.carteira.ObterOuCriarCarteira(idRede, idUsuario, strings.TrimSpace(rede.MoedaVirtualNome), cotacao); err != nil {
		return nil, nil, err
	}
	saldo, err := s.carteira.ObterSaldoToken(idRede, idUsuario)
	if err != nil {
		return nil, nil, err
	}
	if saldo+1e-9 < valorToken {
		return nil, nil, repositorios.ErrSaldoInsuficiente
	}

	restante := round2(calc.ValorFinal - valorMoedaFiat)
	if restante < 0 {
		restante = 0
	}
	meioRestante = strings.ToUpper(strings.TrimSpace(meioRestante))

	modo := NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo)
	var postoCompra *string
	if modo == modelos.GatewayPagamentoModoPosto {
		p := strings.TrimSpace(idPosto)
		if p == "" {
			return nil, nil, errors.New("selecione o posto em que vai abastecer")
		}
		postoCompra = &p
	} else if strings.TrimSpace(idPosto) != "" {
		p := strings.TrimSpace(idPosto)
		postoCompra = &p
	}

	idCompra := uuid.New().String()
	agoraDeb := agora
	regBase := func() *repositorios.VoucherCompraRegistro {
		reg := &repositorios.VoucherCompraRegistro{
			ID:               idCompra,
			RedeID:           idRede,
			UsuarioID:        idUsuario,
			ValorSolicitado:  calc.ValorSolicitado,
			DescontoAplicado: 0,
			ValorFinal:       calc.ValorFinal,
			TipoBeneficio:    "DESCONTO",
			PostoCompraID:    postoCompra,
			ValorMoedaFiat:   valorMoedaFiat,
			ValorMoedaToken:  valorToken,
			MoedaDebitadaEm:  &agoraDeb,
		}
		if calc.Litros != nil {
			v := *calc.Litros
			reg.Litros = &v
		}
		if idCombustivelRede != nil && strings.TrimSpace(*idCombustivelRede) != "" {
			sid := strings.TrimSpace(*idCombustivelRede)
			reg.CombustivelRedeID = &sid
		}
		return reg
	}

	// 100% moeda → ATIVO imediato.
	if restante < 0.005 {
		reg := regBase()
		reg.Status = "ATIVO"
		reg.MeioPagamento = modelos.MeioPagamentoMoedaVirtual
		cod := gerarCodigoResgate()
		reg.CodigoResgate = &cod
		expR := s.expiraResgateAposPagamentoAprovado(idRede, agora)
		reg.ExpiraResgate = &expR

		if err := s.debitarMoedaVoucher(idRede, idUsuario, valorToken, idCompra); err != nil {
			return nil, nil, err
		}
		var lastErr error
		for range 8 {
			lastErr = s.repo.CriarAtivoMoedaVirtual(reg)
			if lastErr == nil {
				log.Printf("voucher_moeda 100%%: rede=%s compra=%s codigo=%s fiat=%.2f", idRede, reg.ID, cod, valorMoedaFiat)
				s.registrarEventoVoucher(modelos.EventoVoucherGerado, reg, "")
				return reg, nil, nil
			}
			if !strings.Contains(strings.ToLower(lastErr.Error()), "unique") &&
				!strings.Contains(strings.ToLower(lastErr.Error()), "duplicate") {
				return nil, nil, lastErr
			}
			cod = gerarCodigoResgate()
			reg.CodigoResgate = &cod
		}
		return nil, nil, fmt.Errorf("falha ao gerar codigo unico: %w", lastErr)
	}

	// Parcial: restante PIX ou DINHEIRO.
	if meioRestante == "" {
		return nil, nil, errors.New("informe meio_pagamento do restante (PIX ou DINHEIRO)")
	}
	if meioRestante != modelos.MeioPagamentoPix && meioRestante != modelos.MeioPagamentoDinheiro {
		return nil, nil, errors.New("restante deve ser PIX ou DINHEIRO")
	}
	if restante < 1.0 {
		return nil, nil, errors.New("restante apos moeda deve ser zero ou pelo menos R$ 1,00")
	}

	if meioRestante == modelos.MeioPagamentoDinheiro {
		if !meios.Dinheiro {
			return nil, nil, errors.New("este posto/rede nao aceita dinheiro para o restante")
		}
		reg := regBase()
		reg.Status = "AGUARDANDO_DINHEIRO"
		reg.MeioPagamento = modelos.MeioPagamentoDinheiro
		cod := gerarCodigoResgate()
		reg.CodigoResgate = &cod
		expR := s.expiraResgateAposPagamentoAprovado(idRede, agora)
		reg.ExpiraResgate = &expR

		if err := s.debitarMoedaVoucher(idRede, idUsuario, valorToken, idCompra); err != nil {
			return nil, nil, err
		}
		var lastErr error
		for range 8 {
			lastErr = s.repo.CriarAguardandoDinheiro(reg)
			if lastErr == nil {
				log.Printf("voucher_moeda+dinheiro: rede=%s compra=%s restante=%.2f", idRede, reg.ID, restante)
				s.registrarEventoVoucher(modelos.EventoVoucherGerado, reg, "")
				return reg, nil, nil
			}
			if !strings.Contains(strings.ToLower(lastErr.Error()), "unique") &&
				!strings.Contains(strings.ToLower(lastErr.Error()), "duplicate") {
				return nil, nil, lastErr
			}
			cod = gerarCodigoResgate()
			reg.CodigoResgate = &cod
		}
		return nil, nil, fmt.Errorf("falha ao gerar codigo unico: %w", lastErr)
	}

	// Restante PIX.
	if !meios.Pix {
		return nil, nil, errors.New("este posto/rede nao aceita PIX para o restante")
	}
	if NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo) == modelos.GatewayPagamentoModoPosto &&
		strings.TrimSpace(idPosto) == "" {
		return nil, nil, errors.New("selecione o posto em que vai abastecer")
	}
	gw, err := ResolverGatewayPagamento(s.rede, s.mpGW, s.eredeGW, s.posto, s.cfg, idRede, idPosto)
	if err != nil {
		return nil, nil, err
	}
	ref := prefixoRefVoucherCompra + idCompra
	expP := agora.Add(s.duracaoPagamentoPix(idRede))
	notifURL := gw.MpWebhookURL
	if gw.Provedor == modelos.GatewayProvedorERede {
		notifURL = gw.ERedeWebhookURL
	}
	res, err := CriarPixVoucher(ctx, gw, CriarPixVoucherInput{
		Valor:           restante,
		Referencia:      ref,
		PayerEmail:      payerEmail,
		DocTipo:         docTipo,
		DocNumero:       docNumero,
		NotificationURL: notifURL,
	}, expP)
	if err != nil {
		return nil, nil, err
	}

	if err := s.debitarMoedaVoucher(idRede, idUsuario, valorToken, idCompra); err != nil {
		return nil, res, err
	}

	reg := regBase()
	reg.Status = "AGUARDANDO_PAGAMENTO"
	reg.MeioPagamento = modelos.MeioPagamentoPix
	reg.GatewayProvedor = gw.Provedor
	reg.ReferenciaPagamento = &ref
	reg.ExpiraPagamento = &expP
	reg.PostoCompraID = gw.PostoIDCompra
	if tid := strings.TrimSpace(res.IDExterno); tid != "" {
		reg.GatewayTID = &tid
	}
	if res.PaymentIDNumerico > 0 {
		mpid := res.PaymentIDNumerico
		reg.MpPaymentID = &mpid
	}
	if err := s.repo.CriarPendenteComPix(reg); err != nil {
		return nil, res, err
	}
	logPixVoucherCriado(idRede, reg, gw, res)
	log.Printf("voucher_moeda+pix: rede=%s compra=%s moeda=%.2f pix=%.2f", idRede, reg.ID, valorMoedaFiat, restante)
	s.registrarEventoVoucher(modelos.EventoVoucherGerado, reg, "")
	return reg, res, nil
}

func (s *ServicoVoucherCompra) debitarMoedaVoucher(idRede, idUsuario string, valorToken float64, compraID string) error {
	return s.carteira.DebitarMoeda(idRede, idUsuario, valorToken, tipoRefVoucherMoeda, compraID)
}

// UsouMoedaVirtual true se a compra debitou moeda (sem Indique 1ª compra / promo).
func UsouMoedaVirtual(vc *repositorios.VoucherCompraRegistro) bool {
	return vc != nil && vc.ValorMoedaFiat > 1e-9
}

// ValorRestanteACobrar valor ainda devido no posto/PIX após moeda.
func ValorRestanteACobrar(vc *repositorios.VoucherCompraRegistro) float64 {
	if vc == nil {
		return 0
	}
	r := round2(vc.ValorFinal - vc.ValorMoedaFiat)
	if r < 0 {
		return 0
	}
	return r
}

// RespostaMoedaVoucherJSON resposta de compra 100% moeda (já ATIVO).
func RespostaMoedaVoucherJSON(reg *repositorios.VoucherCompraRegistro) map[string]any {
	cod := ""
	if reg.CodigoResgate != nil {
		cod = strings.TrimSpace(*reg.CodigoResgate)
	}
	return map[string]any{
		"compra_id":         reg.ID,
		"status":            reg.Status,
		"meio_pagamento":    reg.MeioPagamento,
		"valor_final":       reg.ValorFinal,
		"valor_moeda_fiat":  reg.ValorMoedaFiat,
		"valor_moeda_token": reg.ValorMoedaToken,
		"valor_restante":    0.0,
		"tipo_beneficio":    reg.TipoBeneficio,
		"codigo_resgate":    cod,
		"expira_resgate_em": reg.ExpiraResgate,
		"aviso_titulo":      "Voucher ativo",
		"aviso_corpo":       "Pagamento integral com moeda virtual. Apresente o codigo no posto.",
	}
}

// anexarMoedaNaResposta inclui campos de moeda em respostas PIX/dinheiro.
func anexarMoedaNaResposta(out map[string]any, reg *repositorios.VoucherCompraRegistro) {
	if out == nil || reg == nil || reg.ValorMoedaFiat <= 0 {
		return
	}
	out["valor_moeda_fiat"] = reg.ValorMoedaFiat
	out["valor_moeda_token"] = reg.ValorMoedaToken
	out["valor_restante"] = ValorRestanteACobrar(reg)
}
