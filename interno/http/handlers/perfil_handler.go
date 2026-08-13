package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

func (h *Handlers) PerfilLogado(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getPerfilLogado(w, r)
	case http.MethodPatch, http.MethodPut:
		h.patchPerfilEmailCpf(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "use GET ou PATCH")
	}
}

func (h *Handlers) getPerfilLogado(w http.ResponseWriter, r *http.Request) {
	usuario := middlewares.Usuario(r.Context())
	if usuario == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}

	out := map[string]any{
		"id_usuario":    usuario.IDUsuario,
		"nome_completo": usuario.NomeCompleto,
		"id_rede":       usuario.IDRede,
		"papel":         usuario.Papel,
		"request_id":    middlewares.ObterRequestID(r.Context()),
	}
	email, cpf, err := h.usuarioRedeService.EmailECPFPorUsuarioRede(usuario.IDUsuario, usuario.IDRede)
	if err == nil {
		out["email"] = email
		out["cpf"] = cpf
	} else {
		out["email"] = ""
		out["cpf"] = ""
	}
	if usuario.Papel == modelos.PapelCliente && h.niveisCliente != nil {
		if nc, err := h.niveisCliente.Buscar(usuario.IDRede); err == nil && nc != nil && nc.Ativo {
			out["app_niveis_moeda_ativo"] = true
			cod, err := h.usuarioRedeService.ObterNivelCliente(usuario.IDUsuario, usuario.IDRede)
			if err != nil || strings.TrimSpace(cod) == "" {
				cod = "bronze"
			}
			cod = strings.ToLower(strings.TrimSpace(cod))
			out["nivel_cliente_codigo"] = cod
			out["mult_moeda_nivel_atual"] = h.niveisCliente.FatorMultMoeda(usuario.IDRede, cod)
		}
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

type patchPerfilBody struct {
	Email string `json:"email"`
	CPF   string `json:"cpf"`
}

// patchPerfilEmailCpf PATCH /v1/eu/perfil — grava e-mail e CPF do cliente para PIX.
func (h *Handlers) patchPerfilEmailCpf(w http.ResponseWriter, r *http.Request) {
	usuario := middlewares.Usuario(r.Context())
	if usuario == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}
	if usuario.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "disponivel apenas para clientes")
		return
	}
	var body patchPerfilBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	err := h.usuarioRedeService.AtualizarEmailCpfClienteApp(
		usuario.IDUsuario,
		usuario.IDRede,
		body.Email,
		body.CPF,
	)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErroCliente(w, http.StatusBadRequest, err)
		case errors.Is(err, repositorios.ErrEmailUsuarioEquipeDuplicado):
			utils.ResponderErro(w, http.StatusConflict, "Este e-mail já está em uso nesta rede.")
		case errors.Is(err, repositorios.ErrCPFJaCadastradoNaRede):
			utils.ResponderErro(w, http.StatusConflict, "Este CPF já está em uso nesta rede.")
		case errors.Is(err, repositorios.ErrContaClienteExclusaoNaoAplicada):
			utils.ResponderErro(w, http.StatusNotFound, "Conta não encontrada.")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "Não foi possível salvar o perfil. Tente novamente.")
		}
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	cpf := utils.SomenteDigitosCPF(body.CPF)
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"email": email,
		"cpf":   cpf,
	})
}

// ExcluirContaClienteApp DELETE /v1/eu/conta — encerra conta do cliente (app); anonimiza dados.
func (h *Handlers) ExcluirContaClienteApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}
	if u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "exclusao disponivel apenas para contas de cliente")
		return
	}
	err := h.usuarioRedeService.ExcluirContaClienteApp(u.IDUsuario, u.IDRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrContaClienteExclusaoNaoAplicada):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "nao foi possivel concluir a exclusao")
		}
		return
	}
	h.autenticador.RevogarToken(middlewares.BearerToken(r))
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"mensagem": "conta encerrada"})
}
