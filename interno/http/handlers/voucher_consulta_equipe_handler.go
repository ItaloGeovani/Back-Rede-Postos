package handlers

import (
	"errors"
	"net/http"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// GetVoucherConsultaPorCodigoEquipe GET ?codigo= — frentista / gerente de posto / gestor da rede, mesma rede do token.
func (h *Handlers) GetVoucherConsultaPorCodigoEquipe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "usuario nao autenticado")
		return
	}
	if h.voucherCompraSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	codigo := strings.TrimSpace(r.URL.Query().Get("codigo"))
	if codigo == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe codigo")
		return
	}
	out, err := h.voucherCompraSvc.ConsultarPorCodigoResgateEquipe(u.IDRede, codigo)
	if err != nil {
		if errors.Is(err, repositorios.ErrVoucherCompraNaoEncontrado) {
			utils.ResponderErro(w, http.StatusNotFound, "voucher nao encontrado nesta rede")
			return
		}
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao consultar")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}
