package repositorios

import "errors"

var ErrMercadoPagoGatewayNaoConfigurado = errors.New("mercado pago nao configurado para esta rede")
var ErrMercadoPagoGatewayPostoNaoConfigurado = errors.New("mercado pago nao configurado para este posto")

// MercadoPagoGatewayCredenciais credenciais da aplicação Mercado Pago da rede (painel MP).
type MercadoPagoGatewayCredenciais struct {
	AccessToken   string
	WebhookSecret string
}

// PostoMercadoPagoStatus resumo para o painel (modo POSTO).
type PostoMercadoPagoStatus struct {
	PostoID              string `json:"id_posto"`
	Nome                 string `json:"nome"`
	Codigo               string `json:"codigo"`
	MpAccessTokenOK      bool   `json:"mp_access_token_configurado"`
	MpWebhookSecretOK    bool   `json:"mp_webhook_secret_configurado"`
}

// MercadoPagoGatewayRepositorio persiste credenciais MP por rede e por posto.
type MercadoPagoGatewayRepositorio interface {
	BuscarPorRedeID(idRede string) (*MercadoPagoGatewayCredenciais, error)
	Upsert(idRede, accessToken, webhookSecret string) error
	BuscarPorPostoID(idPosto, idRede string) (*MercadoPagoGatewayCredenciais, error)
	UpsertPosto(idPosto, idRede, accessToken, webhookSecret string) error
	ListarStatusPostosPorRede(idRede string) ([]PostoMercadoPagoStatus, error)
}
