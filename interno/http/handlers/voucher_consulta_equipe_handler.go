package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
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
	out, err := h.voucherCompraSvc.ConsultarPorCodigoResgateEquipe(u.IDRede, codigo, u.IDPosto)
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

type bodyVoucherBaixaEquipe struct {
	Codigo         string  `json:"codigo"`
	IDPosto        *string `json:"id_posto,omitempty"`
	OperadorCodigo string  `json:"operador_codigo,omitempty"`
	OperadorSenha  string  `json:"operador_senha,omitempty"`
}

// PostVoucherBaixaEquipe POST JSON { codigo, id_posto?, operador_codigo?, operador_senha? } — registra uso (USADO); frentista/gerente usam posto do token; gestor pode enviar id_posto.
func (h *Handlers) PostVoucherBaixaEquipe(w http.ResponseWriter, r *http.Request) {
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
	var body bodyVoucherBaixaEquipe
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "corpo json invalido")
		return
	}
	out, err := h.voucherCompraSvc.RegistrarBaixaPorCodigoEquipe(u, body.Codigo, body.IDPosto, body.OperadorCodigo, body.OperadorSenha)
	if err != nil {
		if errors.Is(err, repositorios.ErrVoucherCompraNaoEncontrado) {
			utils.ResponderErro(w, http.StatusNotFound, "voucher nao encontrado nesta rede")
			return
		}
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, servicos.ErrVoucherEquipeSemPosto) ||
			errors.Is(err, servicos.ErrVoucherEquipePapelBaixa) ||
			errors.Is(err, servicos.ErrVoucherEquipeOperadorObrigatorio) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, servicos.ErrVoucherEquipeOperadorInvalido) {
			utils.ResponderErro(w, http.StatusUnauthorized, err.Error())
			return
		}
		if errors.Is(err, servicos.ErrVoucherEquipeNaoAtivoUso) ||
			errors.Is(err, servicos.ErrVoucherEquipeResgateExpirado) ||
			errors.Is(err, servicos.ErrVoucherEquipePostoIncorreto) ||
			errors.Is(err, repositorios.ErrVoucherBaixaNaoPermitida) {
			utils.ResponderErro(w, http.StatusConflict, err.Error())
			return
		}
		log.Printf("voucher baixa equipe: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao registrar uso do voucher")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}
