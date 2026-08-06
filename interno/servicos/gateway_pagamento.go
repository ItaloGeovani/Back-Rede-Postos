package servicos

import (
	"errors"
	"strings"

	"gaspass-servidor/interno/config"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

// GatewayContext credenciais resolvidas para criar cobrança PIX.
type GatewayContext struct {
	Provedor        string
	Modo            string
	PostoIDCompra   *string
	Meios           modelos.GatewayMeiosHabilitados
	// Mercado Pago
	MpAccessToken   string
	MpWebhookSecret string
	MpWebhookURL    string
	// e.Rede
	ERedePV           string
	ERedeClientSecret string
	ERedeAmbiente     string
	ERedeWebhookURL   string
}

// ResolverGatewayPagamento escolhe provedor, modo e credenciais.
func ResolverGatewayPagamento(
	redeRepo repositorios.RedeRepositorio,
	mpGW repositorios.MercadoPagoGatewayRepositorio,
	eredeGW repositorios.ERedeGatewayRepositorio,
	postoRepo interface {
		BuscarPorIDNaRede(idPosto, idRede string) (*modelos.Posto, error)
	},
	cfg config.Config,
	idRede string,
	idPostoRequisicao string,
) (*GatewayContext, error) {
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

	provedor := NormalizarGatewayProvedorAtivo(rede.GatewayProvedorAtivo)
	meios := rede.GatewayMeiosHabilitados
	idPosto := strings.TrimSpace(idPostoRequisicao)
	if idPosto != "" && postoRepo != nil {
		posto, errP := postoRepo.BuscarPorIDNaRede(idPosto, idRede)
		if errP != nil {
			return nil, errP
		}
		meios = modelos.IntersecaoMeios(rede.GatewayMeiosHabilitados, posto.GatewayMeiosHabilitados)
	}
	if !meios.Pix {
		if idPosto != "" {
			return nil, errors.New("este posto nao aceita pagamento pix no momento")
		}
		return nil, errors.New("rede nao aceita pagamento pix no momento")
	}

	out := &GatewayContext{
		Provedor: provedor,
		Modo:     modo,
		Meios:    meios,
	}

	if modo == modelos.GatewayPagamentoModoPosto {
		if idPosto == "" {
			return nil, errors.New("informe o posto para pagamento (modo gateway por posto)")
		}
		out.PostoIDCompra = &idPosto
		if provedor == modelos.GatewayProvedorERede {
			return resolverERedePosto(eredeGW, out, idPosto, idRede, base)
		}
		return resolverMPPosto(mpGW, out, idPosto, idRede, base)
	}

	if provedor == modelos.GatewayProvedorERede {
		return resolverERedeRede(eredeGW, out, idRede, base)
	}
	return resolverMPRede(mpGW, out, idRede, base)
}

func resolverMPRede(mpGW repositorios.MercadoPagoGatewayRepositorio, out *GatewayContext, idRede, base string) (*GatewayContext, error) {
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
	out.MpAccessToken = creds.AccessToken
	out.MpWebhookSecret = creds.WebhookSecret
	out.MpWebhookURL = base + "/v1/public/mercadopago/webhook/" + idRede
	return out, nil
}

func resolverMPPosto(mpGW repositorios.MercadoPagoGatewayRepositorio, out *GatewayContext, idPosto, idRede, base string) (*GatewayContext, error) {
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
	out.MpAccessToken = creds.AccessToken
	out.MpWebhookSecret = creds.WebhookSecret
	out.MpWebhookURL = base + "/v1/public/mercadopago/webhook/" + idRede + "/" + idPosto
	return out, nil
}

func resolverERedeRede(eredeGW repositorios.ERedeGatewayRepositorio, out *GatewayContext, idRede, base string) (*GatewayContext, error) {
	creds, err := eredeGW.BuscarPorRedeID(idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrERedeGatewayNaoConfigurado) {
			return nil, errors.New("rede sem e.rede configurado")
		}
		return nil, err
	}
	if strings.TrimSpace(creds.PV) == "" || strings.TrimSpace(creds.ClientSecret) == "" {
		return nil, errors.New("rede sem pv/token e.rede")
	}
	out.ERedePV = creds.PV
	out.ERedeClientSecret = creds.ClientSecret
	out.ERedeAmbiente = creds.Ambiente
	out.ERedeWebhookURL = base + "/v1/public/erede/webhook/" + idRede
	return out, nil
}

func resolverERedePosto(eredeGW repositorios.ERedeGatewayRepositorio, out *GatewayContext, idPosto, idRede, base string) (*GatewayContext, error) {
	creds, err := eredeGW.BuscarPorPostoID(idPosto, idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrERedeGatewayPostoNaoConfigurado) {
			return nil, errors.New("posto sem e.rede configurado")
		}
		return nil, err
	}
	if strings.TrimSpace(creds.PV) == "" || strings.TrimSpace(creds.ClientSecret) == "" {
		return nil, errors.New("posto sem pv/token e.rede")
	}
	out.ERedePV = creds.PV
	out.ERedeClientSecret = creds.ClientSecret
	out.ERedeAmbiente = creds.Ambiente
	out.ERedeWebhookURL = base + "/v1/public/erede/webhook/" + idRede + "/" + idPosto
	return out, nil
}

func NormalizarGatewayPagamentoModo(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == modelos.GatewayPagamentoModoPosto {
		return modelos.GatewayPagamentoModoPosto
	}
	return modelos.GatewayPagamentoModoRede
}
