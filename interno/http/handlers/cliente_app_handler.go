package handlers

import (
	"errors"
	"net/http"
	"strings"

	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type reqCadastroClienteApp struct {
	IDRede         string `json:"id_rede"`
	Usuario        string `json:"usuario"`
	NomeCompleto   string `json:"nome_completo"` // alias: tratado como usuario
	Senha          string `json:"senha"`
	ConfirmarSenha string `json:"confirmar_senha"`
	Telefone       string `json:"telefone"`
	// Indique e ganhe: codigo de quem indicou (opcional).
	CodigoIndicacao string `json:"codigo_indicacao"`
}

// PublicCadastroClienteApp POST /v1/public/clientes/cadastro — cadastro de cliente no app (sem auth).
// Novos cadastros: usuario (a-z0-9) + senha. Login legado por e-mail continua no login-painel.
func (h *Handlers) PublicCadastroClienteApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqCadastroClienteApp
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}

	token, sessao, err := h.usuarioRedeService.CadastrarClienteApp(servicos.CadastroClienteAppInput{
		IDRede:          strings.TrimSpace(req.IDRede),
		Usuario:         strings.TrimSpace(req.Usuario),
		NomeCompleto:    strings.TrimSpace(req.NomeCompleto),
		Senha:           req.Senha,
		ConfirmarSenha:  req.ConfirmarSenha,
		Telefone:        req.Telefone,
		CodigoIndicacao: req.CodigoIndicacao,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrUsuarioClienteDuplicado):
			utils.ResponderErro(w, http.StatusConflict, "usuario ja cadastrado nesta rede")
		case errors.Is(err, repositorios.ErrEmailUsuarioEquipeDuplicado):
			utils.ResponderErro(w, http.StatusConflict, "usuario ja cadastrado nesta rede")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "nao foi possivel concluir o cadastro")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "cadastro realizado com sucesso",
		"token":    token,
		"sessao":   sessao,
	})
}

// PublicUsuarioClienteDisponivel GET /v1/public/clientes/usuario-disponivel?id_rede=&usuario=
func (h *Handlers) PublicUsuarioClienteDisponivel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "use GET")
		return
	}
	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	usuario := strings.TrimSpace(r.URL.Query().Get("usuario"))
	ok, err := h.usuarioRedeService.UsuarioClienteDisponivel(idRede, usuario)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao verificar usuario")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"usuario":    utils.NormalizarUsuarioCliente(usuario),
		"disponivel": ok,
	})
}
