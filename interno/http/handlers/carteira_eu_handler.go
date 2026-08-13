package handlers

import (
	"net/http"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
)

// GetEuCarteiraSaldo GET /v1/eu/carteira/saldo — saldo em token (moeda virtual) do cliente.
func (h *Handlers) GetEuCarteiraSaldo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.carteiraRepo == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes do app")
		return
	}
	rede := strings.TrimSpace(u.IDRede)
	uid := strings.TrimSpace(u.IDUsuario)
	if rede == "" || uid == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "sessao sem rede")
		return
	}
	saldo, err := h.carteiraRepo.ObterSaldoToken(rede, uid)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao obter saldo")
		return
	}
	red, errR := h.redeService.BuscarPorID(rede)
	m := map[string]any{
		"saldo_token": saldo,
	}
	if errR == nil && red != nil {
		m["moeda_virtual_nome"] = strings.TrimSpace(red.MoedaVirtualNome)
		dias := red.MoedaVirtualExpiraDias
		m["moeda_virtual_expira_dias"] = dias
		m["expiracao_ativa"] = dias > 0
	}
	utils.ResponderJSON(w, http.StatusOK, m)
}

// GetEuCarteiraLotes GET /v1/eu/carteira/lotes — lotes ativos com tempo de expiração.
func (h *Handlers) GetEuCarteiraLotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.carteiraRepo == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes do app")
		return
	}
	rede := strings.TrimSpace(u.IDRede)
	uid := strings.TrimSpace(u.IDUsuario)
	if rede == "" || uid == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "sessao sem rede")
		return
	}
	lotes, err := h.carteiraRepo.ListarLotesAtivos(rede, uid)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar lotes")
		return
	}
	nome := ""
	dias := 0
	if red, errR := h.redeService.BuscarPorID(rede); errR == nil && red != nil {
		nome = strings.TrimSpace(red.MoedaVirtualNome)
		dias = red.MoedaVirtualExpiraDias
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens":                      lotes,
		"total":                      len(lotes),
		"moeda_virtual_nome":         nome,
		"moeda_virtual_expira_dias":  dias,
		"expiracao_ativa":            dias > 0,
	})
}
