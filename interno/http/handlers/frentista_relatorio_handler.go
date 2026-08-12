package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type bodyRelatorioMeusFrentista struct {
	OperadorCodigo string `json:"operador_codigo"`
	OperadorSenha  string `json:"operador_senha"`
	Periodo        string `json:"periodo"`
}

// PostRelatorioMeusFrentista POST { operador_codigo, operador_senha, periodo: hoje|7d }
// Relatório de baixas do frentista autenticado no modal (não o da sessão do PC).
func (h *Handlers) PostRelatorioMeusFrentista(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	var body bodyRelatorioMeusFrentista
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "corpo json invalido")
		return
	}
	out, err := h.voucherCompraSvc.RelatorioBaixasFrentista(u, body.OperadorCodigo, body.OperadorSenha, body.Periodo)
	if err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) ||
			errors.Is(err, servicos.ErrVoucherEquipeOperadorObrigatorio) ||
			errors.Is(err, servicos.ErrVoucherEquipeSemPosto) ||
			errors.Is(err, servicos.ErrVoucherEquipePapelBaixa) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, servicos.ErrVoucherEquipeOperadorInvalido) {
			utils.ResponderErro(w, http.StatusUnauthorized, err.Error())
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar relatorio")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}
