package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// PostVoucherCompraCalcular POST /v1/eu/vouchers/calcular
func (h *Handlers) PostVoucherCompraCalcular(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes")
		return
	}
	if h.voucherCompraSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	var body struct {
		Valor                 float64  `json:"valor"`
		IDCampanha            *string  `json:"id_campanha"`
		IDCombustivelRede     *string  `json:"id_combustivel_rede"`
		Litros                *float64 `json:"litros"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	out, err := h.voucherCompraSvc.Calcular(u.IDRede, body.Valor, body.IDCampanha, time.Now(), body.IDCombustivelRede, body.Litros)
	if err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, "informe um valor minimo de R$ 1,00 e verifique a campanha")
			return
		}
		if errors.Is(err, servicos.ErrVoucherCampanhaInvalida) {
			utils.ResponderErro(w, http.StatusBadRequest, "campanha invalida")
			return
		}
		utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

// PostVoucherCompraPagar POST /v1/eu/vouchers/pagar — cria cobrança PIX e registro.
func (h *Handlers) PostVoucherCompraPagar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes")
		return
	}
	if h.voucherCompraSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	var body struct {
		Valor             float64  `json:"valor"`
		IDCampanha        *string  `json:"id_campanha"`
		IDCombustivelRede *string  `json:"id_combustivel_rede"`
		Litros            *float64 `json:"litros"`
		IDPosto           string   `json:"id_posto"`
		PayerEmail        string   `json:"payer_email"`
		DocTipo           string   `json:"doc_tipo"`
		DocNumero         string   `json:"doc_numero"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	body.PayerEmail = strings.TrimSpace(body.PayerEmail)
	body.DocTipo = strings.TrimSpace(body.DocTipo)
	body.DocNumero = strings.TrimSpace(body.DocNumero)
	if body.PayerEmail == "" || !strings.Contains(body.PayerEmail, "@") {
		utils.ResponderErro(w, http.StatusBadRequest, "payer_email invalido")
		return
	}
	if body.DocTipo == "" || body.DocNumero == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "doc_tipo e doc_numero obrigatorios")
		return
	}
	ctx := r.Context()
	reg, pay, err := h.voucherCompraSvc.PagarComPixInicia(ctx, u.IDRede, u.IDUsuario, body.Valor, body.IDCampanha, body.IDCombustivelRede, body.Litros, strings.TrimSpace(body.IDPosto), body.PayerEmail, body.DocTipo, body.DocNumero, time.Now())
	if err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) || errors.Is(err, servicos.ErrVoucherCampanhaInvalida) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		return
	}
	qr, qrB64 := servicos.ExtrairQRPixDoPagamento(pay)
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"compra_id":        reg.ID,
		"status":            reg.Status,
		"payment_id":        pay.ID,
		"valor_final":       reg.ValorFinal,
		"tipo_beneficio":    reg.TipoBeneficio,
		"cashback_percentual": reg.CashbackPercentual,
		"cashback_previsto": reg.CashbackValor,
		"expira_pagamento":  reg.ExpiraPagamento,
		"qr_code":          qr,
		"qr_code_base64":   qrB64,
		"mp_status":        pay.Status,
	})
}

// GetVoucherCompraPixRetomar GET /v1/eu/vouchers/pix-retomar?id= — QR/copia-e-cola a partir de compra ainda pendente (requery MP).
func (h *Handlers) GetVoucherCompraPixRetomar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id")
		return
	}
	if h.voucherCompraSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	reg, pay, err := h.voucherCompraSvc.RetomarDadosPixPendente(r.Context(), id, u.IDRede, u.IDUsuario)
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		return
	}
	qr, qrB64 := servicos.ExtrairQRPixDoPagamento(pay)
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"compra_id":        reg.ID,
		"status":            reg.Status,
		"payment_id":        pay.ID,
		"valor_final":       reg.ValorFinal,
		"tipo_beneficio":    reg.TipoBeneficio,
		"cashback_percentual": reg.CashbackPercentual,
		"cashback_previsto": reg.CashbackValor,
		"expira_pagamento":  reg.ExpiraPagamento,
		"qr_code":           qr,
		"qr_code_base64":   qrB64,
		"mp_status":         pay.Status,
	})
}

// GetVoucherCompras GET /v1/eu/vouchers
func (h *Handlers) GetVoucherCompras(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes")
		return
	}
	if h.voucherCompraSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	lista, err := h.voucherCompraSvc.ListarMeus(u.IDRede, u.IDUsuario)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"itens": lista, "total": len(lista)})
}

// GetVoucherCompraUsosCampanha GET /v1/eu/vouchers/usos-campanha — usos com pagamento aprovado (ATIVO/USADO) por id de campanha.
func (h *Handlers) GetVoucherCompraUsosCampanha(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes")
		return
	}
	if h.voucherCompraSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	m, err := h.voucherCompraSvc.UsosAprovadosPorCampanha(u.IDRede, u.IDUsuario)
	if err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao contar usos")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"usos_por_campanha": m})
}

// GetVoucherCompraDetalhe GET /v1/eu/vouchers/detalhe?id=uuid
func (h *Handlers) GetVoucherCompraDetalhe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id")
		return
	}
	if h.voucherCompraSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	v, err := h.voucherCompraSvc.BuscarMeu(id, u.IDRede, u.IDUsuario)
	if err != nil {
		if errors.Is(err, repositorios.ErrVoucherCompraNaoEncontrado) {
			utils.ResponderErro(w, http.StatusNotFound, "nao encontrado")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"voucher": v})
}
