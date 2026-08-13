package handlers

import (
	"errors"
	"net/http"

	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type reqCriarAdmin struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type reqEditarAdmin struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Ativo bool   `json:"ativo"`
}

type reqLoginAdmin struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

func (h *Handlers) CriarAdministradorGeralDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqCriarAdmin
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}

	admin, err := h.adminService.Criar(req.Nome, req.Email, req.Senha)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrEmailJaCadastrado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao criar administrador geral")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "administrador geral criado com sucesso",
		"admin":    admin,
	})
}

func (h *Handlers) EditarAdministradorGeralDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqEditarAdmin
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}

	admin, err := h.adminService.Editar(req.ID, req.Nome, req.Email, req.Ativo)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrEmailJaCadastrado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		case errors.Is(err, repositorios.ErrAdminNaoEncontrado):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao editar administrador geral")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "administrador geral atualizado com sucesso",
		"admin":    admin,
	})
}

func (h *Handlers) LoginAdministradorGeralDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqLoginAdmin
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}

	token, sessao, err := h.adminService.Login(req.Email, req.Senha)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, servicos.ErrCredenciais):
			utils.ResponderErro(w, http.StatusUnauthorized, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao autenticar administrador")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "login de administrador realizado com sucesso",
		"token":    token,
		"sessao":   sessao,
	})
}
