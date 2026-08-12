package modelos

// UsuarioVinculoRede representa um usuario (qualquer papel) vinculado a uma rede.
type UsuarioVinculoRede struct {
	ID       string `json:"id"`
	IDRede   string `json:"id_rede"`
	IDPosto  string `json:"id_posto,omitempty"`
	Papel    string `json:"papel"`
	Nome     string `json:"nome"`
	Email    string `json:"email"`
	Codigo   string `json:"codigo,omitempty"`
	Telefone string `json:"telefone"`
	Ativo    bool   `json:"ativo"`
}
