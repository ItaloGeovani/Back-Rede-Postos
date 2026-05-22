package servicos

import (
	"errors"
	"strings"

	"gaspass-servidor/interno/config"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

// PagamentoGatewayResolvido credenciais e metadados para criar cobrança PIX.
type PagamentoGatewayResolvido struct {
	Modo          string
	AccessToken   string
	WebhookSecret string
	WebhookURL    string
	PostoIDCompra *string
}

// ResolverGatewayPagamento escolhe credenciais conforme gateway_pagamento_modo da rede.
func ResolverGatewayPagamento(
	redeRepo repositorios.RedeRepositorio,
	mpGW repositorios.MercadoPagoGatewayRepositorio,
	cfg config.Config,
	idRede string,
	idPostoRequisicao string,
) (*PagamentoGatewayResolvido, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrDadosInvalidos
	}
	rede, err := redeRepo.BuscarPorID(idRede)
	if err != nil {
		return nil, err
	}
	modo := strings.ToUpper(strings.TrimSpace(rede.GatewayPagamentoModo))
	if modo != modelos.GatewayPagamentoModoPosto {
		modo = modelos.GatewayPagamentoModoRede
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/")
	if base == "" {
		return nil, errors.New("servidor sem PUBLIC_BASE_URL")
	}

	out := &PagamentoGatewayResolvido{Modo: modo}

	if modo == modelos.GatewayPagamentoModoPosto {
		idPosto := strings.TrimSpace(idPostoRequisicao)
		if idPosto == "" {
			return nil, errors.New("informe o posto para pagamento (modo gateway por posto)")
		}
		creds, err := mpGW.BuscarPorPostoID(idPosto, idRede)
		if err != nil {
			if errors.Is(err, repositorios.ErrMercadoPagoGatewayPostoNaoConfigurado) {
				return nil, errors.New("posto sem mercado pago configurado")
			}
			return nil, err
		}
		if strings.TrimSpace(creds.AccessToken) == "" {
			return nil, errors.New("posto sem mp_access_token")
		}
		out.AccessToken = creds.AccessToken
		out.WebhookSecret = creds.WebhookSecret
		out.WebhookURL = base + "/v1/public/mercadopago/webhook/" + idRede + "/" + idPosto
		out.PostoIDCompra = &idPosto
		return out, nil
	}

	creds, err := mpGW.BuscarPorRedeID(idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrMercadoPagoGatewayNaoConfigurado) {
			return nil, errors.New("rede sem mercado pago configurado")
		}
		return nil, err
	}
	if strings.TrimSpace(creds.AccessToken) == "" {
		return nil, errors.New("rede sem mp_access_token")
	}
	out.AccessToken = creds.AccessToken
	out.WebhookSecret = creds.WebhookSecret
	out.WebhookURL = base + "/v1/public/mercadopago/webhook/" + idRede
	return out, nil
}

func NormalizarGatewayPagamentoModo(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == modelos.GatewayPagamentoModoPosto {
		return modelos.GatewayPagamentoModoPosto
	}
	return modelos.GatewayPagamentoModoRede
}
