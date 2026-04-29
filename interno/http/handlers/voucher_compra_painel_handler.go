package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// GetVouchersComprasPainelEquipe GET ?limite=&offset=&status= — JWT define a rede.
func (h *Handlers) GetVouchersComprasPainelEquipe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}
	idRede := strings.TrimSpace(u.IDRede)
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "rede nao associada ao usuario")
		return
	}
	h.responderVouchersComprasPainel(w, r, idRede)
}

// GetVouchersComprasPainelAdmin GET ?id_rede=&limite=&offset=&status= — super-admin.
func (h *Handlers) GetVouchersComprasPainelAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	h.responderVouchersComprasPainel(w, r, idRede)
}

func (h *Handlers) responderVouchersComprasPainel(w http.ResponseWriter, r *http.Request, idRede string) {
	if h.voucherCompraSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	limite := 50
	if v := strings.TrimSpace(r.URL.Query().Get("limite")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limite = n
		}
	}
	offset := 0
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	itens, total, err := h.voucherCompraSvc.ListarPainelPorRede(idRede, limite, offset, status)
	if err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar vouchers")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens": itens,
		"total": total,
	})
}
