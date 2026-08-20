package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
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
		log.Printf("presenca-app listar: rede=%s err=%v", idRede, err)
		utils.ResponderErro(w, http.StatusInternalServerError, "Não foi possível listar a presença dos clientes.")
		return
	}
	if itens == nil {
		itens = []repositorios.ClientePresencaAppItem{}
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"total_clientes":                totalC,
		"total_com_presenca_registrada": totalP,
		"limite":                        lim,
		"minutos_online":                minOn,
		"itens":                         itens,
	})
}

// ListarClientesCarteiraPainel GET /v1/.../clientes/carteira — ranking de clientes por saldo da moeda.
func (h *Handlers) ListarClientesCarteiraPainel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	idRede := ""
	u := middlewares.Usuario(r.Context())
	if u != nil && (u.Papel == modelos.PapelGestorRede || u.Papel == modelos.PapelGerentePosto) {
		var ok bool
		idRede, ok = h.idRedeDaSessao(w, r)
		if !ok {
			return
		}
	} else {
		// Super-admin: id_rede obrigatório na query.
		idRede = strings.TrimSpace(r.URL.Query().Get("id_rede"))
		if idRede == "" {
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
			return
		}
	}

	lim := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limite")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			lim = n
		}
	}
	off := 0
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			off = n
		}
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	ordenar := strings.TrimSpace(r.URL.Query().Get("ordenar"))
	if ordenar == "" {
		ordenar = "saldo_desc"
	}

	itens, total, moedaNome, err := h.usuarioRedeService.ListarClientesCarteiraRede(idRede, repositorios.ClienteCarteiraFiltro{
		Limite:  lim,
		Offset:  off,
		Q:       q,
		Ordenar: ordenar,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErroCliente(w, http.StatusBadRequest, err)
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "Rede não encontrada.")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "Não foi possível listar os clientes da carteira.")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens":              itens,
		"total":              total,
		"limite":             lim,
		"offset":             off,
		"ordenar":            ordenar,
		"moeda_virtual_nome": moedaNome,
	})
}
