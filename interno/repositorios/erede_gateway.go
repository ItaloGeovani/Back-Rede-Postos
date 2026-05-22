package repositorios

import "errors"

var (
	ErrERedeGatewayNaoConfigurado      = errors.New("e.rede nao configurado para esta rede")
	ErrERedeGatewayPostoNaoConfigurado = errors.New("e.rede nao configurado para este posto")
)

// ERedeGatewayCredenciais PV (clientId) + client secret + ambiente.
type ERedeGatewayCredenciais struct {
	PV           string
	ClientSecret string
	Ambiente     string // sandbox | producao
}

// PostoERedeStatus resumo para o painel (modo POSTO).
type PostoERedeStatus struct {
	PostoID           string `json:"id_posto"`
	Nome              string `json:"nome"`
	Codigo            string `json:"codigo"`
	PvConfigurado     bool   `json:"pv_configurado"`
	SecretConfigurado bool   `json:"client_secret_configurado"`
}

// ERedeGatewayRepositorio credenciais e.Rede por rede e por posto.
type ERedeGatewayRepositorio interface {
	BuscarPorRedeID(idRede string) (*ERedeGatewayCredenciais, error)
	Upsert(idRede, pv, clientSecret, ambiente string) error
	BuscarPorPostoID(idPosto, idRede string) (*ERedeGatewayCredenciais, error)
	UpsertPosto(idPosto, idRede, pv, clientSecret, ambiente string) error
	ListarStatusPostosPorRede(idRede string) ([]PostoERedeStatus, error)
}
