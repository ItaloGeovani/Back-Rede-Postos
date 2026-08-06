package modelos

import "time"

type Posto struct {
	ID                      string                  `json:"id"`
	IDRede                  string                  `json:"id_rede"`
	Nome                    string                  `json:"nome"`
	Codigo                  string                  `json:"codigo"`
	NomeFantasia            string                  `json:"nome_fantasia"`
	CNPJ                    string                  `json:"cnpj"`
	LogoURL                 string                  `json:"logo_url"`
	Rua                     string                  `json:"rua"`
	Numero                  string                  `json:"numero"`
	Bairro                  string                  `json:"bairro"`
	Complemento             string                  `json:"complemento"`
	CEP                     string                  `json:"cep"`
	Cidade                  string                  `json:"cidade"`
	Estado                  string                  `json:"estado"`
	Telefone                string                  `json:"telefone"`
	EmailContato            string                  `json:"email_contato"`
	GatewayMeiosHabilitados GatewayMeiosHabilitados `json:"gateway_meios_habilitados"`
	CriadoEm                time.Time               `json:"criado_em"`
	AtualizadoEm            time.Time               `json:"atualizado_em"`
}
