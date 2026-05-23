package servicos

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos/erede"
)

// CriarPixVoucher cria cobrança PIX no provedor configurado.
func CriarPixVoucher(ctx context.Context, gw *GatewayContext, in CriarPixVoucherInput, expiraPagamento time.Time) (*PixCobrancaResult, error) {
	if gw == nil {
		return nil, ErrDadosInvalidos
	}
	if !gw.Meios.Pix {
		return nil, ErrDadosInvalidos
	}
	switch gw.Provedor {
	case modelos.GatewayProvedorERede:
		ref := strings.TrimSpace(in.Referencia)
		if len(ref) > 16 {
			ref = ref[len(ref)-16:]
		}
		ref = strings.TrimPrefix(ref, "vcompra:")
		if ref == "" {
			ref = strings.TrimSpace(in.Referencia)
		}
		pix, err := erede.CriarPixQR(ctx, gw.ERedePV, gw.ERedeClientSecret, gw.ERedeAmbiente, in.Valor, ref, expiraPagamento)
		if err != nil {
			return nil, err
		}
		return &PixCobrancaResult{
			Provedor:     modelos.GatewayProvedorERede,
			IDExterno:    pix.TID,
			QrCode:       pix.QrCodeData,
			QrCodeBase64: pix.QrCodeImage,
			Status:       "pending",
			Referencia:   in.Referencia,
		}, nil
	default:
		res, err := CriarCobrancaPixMercadoPago(ctx, gw.MpAccessToken, CriarCobrancaPixMercadoPagoInput{
			Valor:             in.Valor,
			Descricao:         "Voucher Auto Posto",
			PayerEmail:        in.PayerEmail,
			DocTipo:           in.DocTipo,
			DocNumero:         in.DocNumero,
			ExternalReference: in.Referencia,
			NotificationURL:   in.NotificationURL,
		})
		if err != nil {
			return nil, err
		}
		qr, qrB64 := ExtrairQRPixDoPagamento(res)
		mpID := int64(res.ID)
		return &PixCobrancaResult{
			Provedor:          modelos.GatewayProvedorMercadoPago,
			IDExterno:         strconv.FormatInt(mpID, 10),
			QrCode:            qr,
			QrCodeBase64:      qrB64,
			Status:            strings.TrimSpace(res.Status),
			Referencia:        in.Referencia,
			PaymentIDNumerico: mpID,
		}, nil
	}
}

// ConsultarPixVoucher reconsulta cobrança pendente.
func ConsultarPixVoucher(ctx context.Context, gw *GatewayContext, provedor, gatewayTID string, mpPaymentID *int64) (*PixCobrancaResult, error) {
	if gw == nil {
		return nil, ErrDadosInvalidos
	}
	provedor = NormalizarGatewayProvedorAtivo(provedor)
	switch provedor {
	case modelos.GatewayProvedorERede:
		tid := strings.TrimSpace(gatewayTID)
		if tid == "" {
			return nil, ErrDadosInvalidos
		}
		tx, err := erede.ConsultarTransacaoPorTID(ctx, gw.ERedePV, gw.ERedeClientSecret, gw.ERedeAmbiente, tid)
		if err != nil {
			return nil, err
		}
		st := "pending"
		if erede.TransacaoAprovadaPix(tx) {
			st = "approved"
		}
		ref := strings.TrimSpace(tx.Reference)
		if tx.Authorization != nil && strings.TrimSpace(tx.Authorization.Reference) != "" {
			ref = strings.TrimSpace(tx.Authorization.Reference)
		}
		return &PixCobrancaResult{
			Provedor:           modelos.GatewayProvedorERede,
			IDExterno:          tid,
			Status:             st,
			GatewayStatusLabel: erede.StatusPixLabel(tx),
			Referencia:         ref,
		}, nil
	default:
		if mpPaymentID == nil {
			return nil, errors.New("pagamento nao associado")
		}
		pay, err := ConsultarPagamentoMercadoPago(ctx, gw.MpAccessToken, int(*mpPaymentID))
		if err != nil {
			return nil, err
		}
		qr, qrB64 := ExtrairQRPixDoPagamento(pay)
		mpID := int64(pay.ID)
		return &PixCobrancaResult{
			Provedor:          modelos.GatewayProvedorMercadoPago,
			IDExterno:         strconv.FormatInt(mpID, 10),
			QrCode:            qr,
			QrCodeBase64:      qrB64,
			Status:            strings.TrimSpace(pay.Status),
			Referencia:        strings.TrimSpace(pay.ExternalReference),
			PaymentIDNumerico: mpID,
		}, nil
	}
}
