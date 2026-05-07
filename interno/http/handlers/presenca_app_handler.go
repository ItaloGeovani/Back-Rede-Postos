package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// PostEuAppPresenca POST /v1/eu/app/presenca — heartbeat do app cliente (ultima atividade no sistema).
func (h *Handlers) PostEuAppPresenca(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}
	if u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas contas de cliente do app")
		return
	}
	var body struct {
		Plataforma string `json:"plataforma"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	pl := strings.TrimSpace(strings.ToLower(body.Plataforma))

	err := h.usuarioRedeService.RegistrarPresencaAppCliente(u.IDUsuario, u.IDRede, pl)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, servicos.ErrPresencaClienteNaoAplicavel):
			utils.ResponderErro(w, http.StatusForbidden, "presenca nao aplicavel")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao registrar presenca")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ListarClientesPresencaAppPainel GET /v1/.../clientes/presenca-app — gestor ou gerente: lista clientes com ultima atividade no app.
func (h *Handlers) ListarClientesPresencaAppPainel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	lim := 200
	if v := strings.TrimSpace(r.URL.Query().Get("limite")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			lim = n
		}
	}
	minOn := 15
	if v := strings.TrimSpace(r.URL.Query().Get("minutos_online")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			minOn = n
		}
	}
	totalC, totalP, itens, err := h.usuarioRedeService.ListarPresencaClientesRede(idRede, lim, minOn)
	if err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar presenca")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"total_clientes":                  totalC,
		"total_com_presenca_registrada":   totalP,
		"limite":                          lim,
		"minutos_online":                  minOn,
		"itens":                           itens,
	})
}
