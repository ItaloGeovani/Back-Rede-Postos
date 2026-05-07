package handlers

import (
	"errors"
	"net/http"
	"strings"

	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type reqLoginPainelUnificado struct {
	IDRede string `json:"id_rede"`
	Email  string `json:"email"`
	Senha  string `json:"senha"`
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	// Endpoint inicial para desenvolvimento. A validacao real sera integrada
	// com usuarios no banco e emissao de JWT.
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "autenticacao de desenvolvimento habilitada",
		"tokens_teste": map[string]string{
			"super_admin": "dev-super-admin",
			"gestor_rede": "dev-gestor",
			"frentista":   "dev-frentista",
		},
		"admin_geral_dev": map[string]string{
			"criar":  "POST /v1/admin-geral/dev/criar",
			"login":  "POST /v1/admin-geral/dev/login",
			"editar": "PUT /v1/admin/administradores-gerais/dev/editar",
		},
		"login_painel_unificado": "POST /v1/autenticacao/login-painel (uma requisicao: admin, gestor ou equipe)",
		"gestor_rede_dev": map[string]string{
			"login":            "POST /v1/gestor-rede/dev/login",
			"criar_com_plano":  "POST /v1/admin/gestores-rede/dev/criar",
			"editar_com_plano": "PUT /v1/admin/gestores-rede/dev/editar",
		},
		"redes_dev": map[string]string{
			"listar":         "GET /v1/admin/redes/dev/listar",
			"criar":          "POST /v1/admin/redes/dev/criar",
			"editar":         "PUT /v1/admin/redes/dev/editar",
			"moeda_virtual":  "PATCH /v1/admin/redes/dev/moeda-virtual",
			"ativar":         "PATCH /v1/admin/redes/dev/ativar",
			"desativar":      "PATCH /v1/admin/redes/dev/desativar",
		},
		"uso": "envie Authorization: Bearer <token> nas rotas protegidas e privadas",
	})
}

// LoginPainelUnificadoDev tenta, na ordem: administrador geral, gestor da rede, equipe/cliente no painel.
// Corpo tipico: apenas email e senha; rede e posto vêm do registro encontrado no banco.
// id_rede no JSON e opcional: quando informado, restringe equipe/cliente àquela rede (mesmo email em varias redes).
func (h *Handlers) LoginPainelUnificadoDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqLoginPainelUnificado
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = strings.TrimSpace(req.IDRede)

	token, sessao, err := h.adminService.Login(req.Email, req.Senha)
	if err == nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{
			"mensagem": "login de administrador realizado com sucesso",
			"token":    token,
			"sessao":   sessao,
		})
		return
	}
	if !errors.Is(err, servicos.ErrCredenciais) {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao autenticar administrador")
		}
		return
	}

	token, sessao, err = h.gestorService.Login(req.Email, req.Senha)
	if err == nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{
			"mensagem": "login de gestor da rede realizado com sucesso",
			"token":    token,
			"sessao":   sessao,
		})
		return
	}
	if !errors.Is(err, servicos.ErrCredenciais) {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao autenticar gestor da rede")
		}
		return
	}

	// Email cadastrado como gestor, mas senha inativa/incorreta: não tentar equipe/cliente.
	existeGestor, errGestor := h.gestorService.ExisteGestorPorEmail(req.Email)
	if errGestor != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao autenticar gestor da rede")
		return
	}
	if existeGestor {
		utils.ResponderErro(w, http.StatusUnauthorized, servicos.ErrCredenciais.Error())
		return
	}

	if req.IDRede == "" {
		token, sessao, err = h.usuarioRedeService.LoginPainel(req.Email, req.Senha)
	} else {
		token, sessao, err = h.usuarioRedeService.LoginPainelNaRede(req.Email, req.Senha, req.IDRede)
	}
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, servicos.ErrCredenciais):
			utils.ResponderErro(w, http.StatusUnauthorized, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao autenticar usuario")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "login realizado com sucesso",
		"token":    token,
		"sessao":   sessao,
	})
}
