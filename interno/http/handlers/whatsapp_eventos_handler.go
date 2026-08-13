package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// WhatsAppNotificacoesGestor GET/PUT /v1/gestor-rede/dev/whatsapp-notificacoes
func (h *Handlers) WhatsAppNotificacoesGestor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getWhatsAppNotificacoesGestor(w, r)
	case http.MethodPut:
		h.putWhatsAppNotificacoesGestor(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) getWhatsAppNotificacoesGestor(w http.ResponseWriter, r *http.Request) {
	if h.eventosSvc == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	cfg, err := h.eventosSvc.BuscarConfigWhatsApp(idRede)
	if err != nil {
		log.Printf("whatsapp config get: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	out := *cfg
	out.InstanceTokenMasked = servicos.MascararToken(cfg.InstanceToken)
	out.InstanceToken = ""
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"config":                   out,
		"evolution_base_url_set":   strings.TrimSpace(h.cfg.EvolutionGoBaseURL) != "",
	})
}

func (h *Handlers) putWhatsAppNotificacoesGestor(w http.ResponseWriter, r *http.Request) {
	if h.eventosSvc == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req struct {
		Habilitado          *bool   `json:"habilitado"`
		InstanceName        *string `json:"instance_name"`
		InstanceToken       *string `json:"instance_token"`
		GroupJID            *string `json:"group_jid"`
		NotifyVoucherGerado *bool   `json:"notify_voucher_gerado"`
		NotifyVoucherPago   *bool   `json:"notify_voucher_pago"`
		NotifyVoucherBaixa  *bool   `json:"notify_voucher_baixa"`
		NotifyCampanha      *bool   `json:"notify_campanha"`
	}
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	atual, err := h.eventosSvc.BuscarConfigWhatsApp(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	atual.IDRede = idRede
	if req.Habilitado != nil {
		atual.Habilitado = *req.Habilitado
	}
	if req.InstanceName != nil {
		atual.InstanceName = strings.TrimSpace(*req.InstanceName)
	}
	if req.InstanceToken != nil {
		atual.InstanceToken = strings.TrimSpace(*req.InstanceToken)
	} else {
		atual.InstanceToken = "" // upsert preserva token se vazio
	}
	if req.GroupJID != nil {
		atual.GroupJID = strings.TrimSpace(*req.GroupJID)
	}
	if req.NotifyVoucherGerado != nil {
		atual.NotifyVoucherGerado = *req.NotifyVoucherGerado
	}
	if req.NotifyVoucherPago != nil {
		atual.NotifyVoucherPago = *req.NotifyVoucherPago
	}
	if req.NotifyVoucherBaixa != nil {
		atual.NotifyVoucherBaixa = *req.NotifyVoucherBaixa
	}
	if req.NotifyCampanha != nil {
		atual.NotifyCampanha = *req.NotifyCampanha
	}
	if err := h.eventosSvc.SalvarConfigWhatsApp(atual); err != nil {
		log.Printf("whatsapp config put: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar configuracao")
		return
	}
	salvo, _ := h.eventosSvc.BuscarConfigWhatsApp(idRede)
	if salvo == nil {
		salvo = atual
	}
	out := *salvo
	out.InstanceTokenMasked = servicos.MascararToken(salvo.InstanceToken)
	out.InstanceToken = ""
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "configuracao salva",
		"config":   out,
	})
}

// PostWhatsAppNotificacoesTeste POST /v1/gestor-rede/dev/whatsapp-notificacoes/test
func (h *Handlers) PostWhatsAppNotificacoesTeste(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.eventosSvc == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := h.eventosSvc.EnviarTesteWhatsApp(ctx, idRede); err != nil {
		msg := err.Error()
		log.Printf("whatsapp teste: %v", err)
		utils.ResponderErro(w, http.StatusBadRequest, msg)
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "mensagem de teste enviada",
	})
}

// ListarEventosOperacionaisGestor GET /v1/gestor-rede/dev/eventos-operacionais/listar
func (h *Handlers) ListarEventosOperacionaisGestor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.eventosSvc == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	limite := 50
	if v := strings.TrimSpace(q.Get("limite")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			utils.ResponderErro(w, http.StatusBadRequest, "parametro limite invalido (1 a 200)")
			return
		}
		limite = n
	}
	offset := 0
	if v := strings.TrimSpace(q.Get("offset")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			utils.ResponderErro(w, http.StatusBadRequest, "parametro offset invalido")
			return
		}
		offset = n
	}
	itens, total, err := h.eventosSvc.Listar(idRede, limite, offset)
	if err != nil {
		log.Printf("listar eventos operacionais: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar eventos")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens":  itens,
		"total":  total,
		"limite": limite,
		"offset": offset,
	})
}
