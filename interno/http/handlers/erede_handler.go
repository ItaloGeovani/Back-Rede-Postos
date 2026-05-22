package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/interno/servicos/erede"
	"gaspass-servidor/utils"
)

// ERedeWebhookPublico POST /v1/public/erede/webhook/{rede_id} ou .../{rede_id}/{posto_id}
func (h *Handlers) ERedeWebhookPublico(w http.ResponseWriter, r *http.Request) {
	// Log imediato: qualquer request que chegar neste handler (POST, GET de teste, etc.).
	log.Printf(
		"erede webhook: >>> HIT method=%s path=%q remote=%s content-type=%q ua=%q",
		r.Method, r.URL.Path, r.RemoteAddr, r.Header.Get("Content-Type"), r.UserAgent(),
	)

	if r.Method != http.MethodPost {
		log.Printf("erede webhook: resposta 405 (metodo %s)", r.Method)
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	const prefix = "/v1/public/erede/webhook/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		log.Printf("erede webhook: resposta 404 path=%q", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	parts := strings.Split(rest, "/")
	idRede := ""
	if len(parts) >= 1 {
		idRede = strings.TrimSpace(parts[0])
	}
	idPosto := ""
	if len(parts) >= 2 {
		idPosto = strings.TrimSpace(parts[1])
	}
	if idRede == "" {
		log.Printf("erede webhook: resposta 400 rede_id vazio path=%q", r.URL.Path)
		utils.ResponderErro(w, http.StatusBadRequest, "rede_id invalido")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		log.Printf("erede webhook: resposta 400 leitura corpo rede=%s posto=%s: %v", idRede, idPosto, err)
		utils.ResponderErro(w, http.StatusBadRequest, "corpo invalido")
		return
	}
	_ = r.Body.Close()

	log.Printf(
		"erede webhook: POST body rede=%s posto=%s bytes=%d corpo=%s",
		idRede, idPosto, len(body), truncarLogWebhookERede(body),
	)

	tid, aprovado, err := erede.ParseWebhook(body)
	if err != nil {
		log.Printf("erede webhook: parse rede=%s posto=%s: %v", idRede, idPosto, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if !aprovado {
		log.Printf("erede webhook: ignorado (sem PV.UPDATE_TRANSACTION_PIX) rede=%s posto=%s tid=%s", idRede, idPosto, tid)
		w.WriteHeader(http.StatusOK)
		return
	}

	if h.voucherCompraSvc != nil {
		h.voucherCompraSvc.ProcessarPagamentoAprovadoERede(idRede, tid)
	}
	log.Printf("erede webhook: evento pix aprovado rede=%s posto=%s tid=%s", idRede, idPosto, tid)
	w.WriteHeader(http.StatusOK)
}

// ERedeGatewayGestor GET/PUT /v1/gestor-rede/dev/erede-gateway
func (h *Handlers) ERedeGatewayGestor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getERedeGatewayPainel(w, r)
	case http.MethodPut:
		h.putERedeGatewayPainel(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

// ERedeGatewayPostoGestor PUT /v1/gestor-rede/dev/erede-gateway/posto
func (h *Handlers) ERedeGatewayPostoGestor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || (u.Papel != modelos.PapelGestorRede && u.Papel != modelos.PapelGerentePosto) {
		utils.ResponderErro(w, http.StatusForbidden, "acesso negado")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	rede, err := h.redeService.BuscarPorID(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		return
	}
	if servicos.NormalizarGatewayProvedorAtivo(rede.GatewayProvedorAtivo) != modelos.GatewayProvedorERede {
		utils.ResponderErro(w, http.StatusBadRequest, "rede nao usa e.rede como provedor ativo")
		return
	}
	modo := servicos.NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo)
	if modo != modelos.GatewayPagamentoModoPosto {
		utils.ResponderErro(w, http.StatusBadRequest, "modo por posto necessario para credenciais por unidade")
		return
	}
	var body struct {
		IDPosto      string `json:"id_posto"`
		PV           string `json:"pv"`
		ClientSecret string `json:"client_secret"`
		Ambiente     string `json:"ambiente"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	idPosto := strings.TrimSpace(body.IDPosto)
	if u.Papel == modelos.PapelGerentePosto {
		idPosto = strings.TrimSpace(u.IDPosto)
	}
	if idPosto == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "id_posto obrigatorio")
		return
	}
	if strings.TrimSpace(body.PV) == "" || strings.TrimSpace(body.ClientSecret) == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "pv e client_secret obrigatorios")
		return
	}
	amb := strings.TrimSpace(body.Ambiente)
	if amb == "" {
		amb = "sandbox"
	}
	if err := h.eredeGatewayRepo.UpsertPosto(idPosto, idRede, body.PV, body.ClientSecret, amb); err != nil {
		log.Printf("erede upsert posto=%s: %v", idPosto, err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar")
		return
	}
	wh := h.urlWebhookERedePosto(idRede, idPosto)
	if amb == "sandbox" && wh != "" {
		if err := erede.RegistrarWebhookSandbox(r.Context(), body.PV, body.ClientSecret, amb, wh, "Bearer", ""); err != nil {
			log.Printf("erede webhook sandbox posto=%s rede=%s: %v", idPosto, idRede, err)
		} else {
			log.Printf("erede webhook sandbox registrado posto=%s url=%s", idPosto, wh)
		}
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"webhook_url": wh,
	})
}

// EditarGatewayProvedorGestor PUT /v1/gestor-rede/dev/redes/gateway-provedor
func (h *Handlers) EditarGatewayProvedorGestor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var body struct {
		GatewayProvedorAtivo    string `json:"gateway_provedor_ativo"`
		GatewayMeiosHabilitados struct {
			Pix           bool `json:"pix"`
			CartaoCredito bool `json:"cartao_credito"`
			CartaoDebito  bool `json:"cartao_debito"`
		} `json:"gateway_meios_habilitados"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	meios := modelos.GatewayMeiosHabilitados{
		Pix:           body.GatewayMeiosHabilitados.Pix,
		CartaoCredito: body.GatewayMeiosHabilitados.CartaoCredito,
		CartaoDebito:  body.GatewayMeiosHabilitados.CartaoDebito,
	}
	if !meios.Pix && !meios.CartaoCredito && !meios.CartaoDebito {
		meios.Pix = true
	}
	rede, err := h.redeService.EditarGatewayProvedor(idRede, body.GatewayProvedorAtivo, meios)
	if err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":                      true,
		"gateway_provedor_ativo":  rede.GatewayProvedorAtivo,
		"gateway_meios_habilitados": rede.GatewayMeiosHabilitados,
	})
}

func (h *Handlers) getERedeGatewayPainel(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	u := middlewares.Usuario(r.Context())
	rede, err := h.redeService.BuscarPorID(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		return
	}
	modo := servicos.NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo)
	out := map[string]any{
		"gateway_provedor_ativo":    servicos.NormalizarGatewayProvedorAtivo(rede.GatewayProvedorAtivo),
		"gateway_meios_habilitados": rede.GatewayMeiosHabilitados,
		"gateway_pagamento_modo":    modo,
	}

	if u != nil && u.Papel == modelos.PapelGerentePosto {
		idPosto := strings.TrimSpace(u.IDPosto)
		if modo != modelos.GatewayPagamentoModoPosto {
			out["mensagem"] = "Esta rede usa credenciais e.Rede unicas (configuradas pelo gestor)."
			utils.ResponderJSON(w, http.StatusOK, out)
			return
		}
		out["id_posto"] = idPosto
		out["webhook_url"] = h.urlWebhookERedePosto(idRede, idPosto)
		creds, errC := h.eredeGatewayRepo.BuscarPorPostoID(idPosto, idRede)
		out["pv_configurado"] = false
		out["client_secret_configurado"] = false
		if errC == nil {
			out["pv_configurado"] = strings.TrimSpace(creds.PV) != ""
			out["client_secret_configurado"] = strings.TrimSpace(creds.ClientSecret) != ""
			if strings.TrimSpace(creds.PV) != "" {
				out["pv_mascarado"] = mascararSegredoMercadoPago(creds.PV)
			}
			out["ambiente"] = creds.Ambiente
		}
		utils.ResponderJSON(w, http.StatusOK, out)
		return
	}

	if modo == modelos.GatewayPagamentoModoPosto {
		postos, err := h.eredeGatewayRepo.ListarStatusPostosPorRede(idRede)
		if err != nil {
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar postos")
			return
		}
		base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicBaseURL), "/")
		itens := make([]map[string]any, 0, len(postos))
		for _, p := range postos {
			wh := ""
			if base != "" {
				wh = base + "/v1/public/erede/webhook/" + idRede + "/" + p.PostoID
			}
			itens = append(itens, map[string]any{
				"id_posto":                p.PostoID,
				"nome":                    p.Nome,
				"codigo":                  p.Codigo,
				"webhook_url":             wh,
				"pv_configurado":          p.PvConfigurado,
				"client_secret_configurado": p.SecretConfigurado,
			})
		}
		out["postos"] = itens
		utils.ResponderJSON(w, http.StatusOK, out)
		return
	}
	out["webhook_url"] = h.urlWebhookERede(idRede)
	creds, err := h.eredeGatewayRepo.BuscarPorRedeID(idRede)
	if err != nil {
		out["pv_configurado"] = false
		out["client_secret_configurado"] = false
	} else {
		out["pv_configurado"] = strings.TrimSpace(creds.PV) != ""
		out["client_secret_configurado"] = strings.TrimSpace(creds.ClientSecret) != ""
		if strings.TrimSpace(creds.PV) != "" {
			out["pv_mascarado"] = mascararSegredoMercadoPago(creds.PV)
		}
		out["ambiente"] = creds.Ambiente
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

func (h *Handlers) putERedeGatewayPainel(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	u := middlewares.Usuario(r.Context())
	if u != nil && u.Papel == modelos.PapelGerentePosto {
		utils.ResponderErro(w, http.StatusForbidden, "gerente deve usar PUT erede-gateway/posto")
		return
	}
	rede, err := h.redeService.BuscarPorID(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		return
	}
	if servicos.NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo) != modelos.GatewayPagamentoModoRede {
		utils.ResponderErro(w, http.StatusBadRequest, "no modo por posto use PUT erede-gateway/posto")
		return
	}
	var body struct {
		PV           string `json:"pv"`
		ClientSecret string `json:"client_secret"`
		Ambiente     string `json:"ambiente"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	if strings.TrimSpace(body.PV) == "" || strings.TrimSpace(body.ClientSecret) == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "pv e client_secret obrigatorios")
		return
	}
	amb := strings.TrimSpace(body.Ambiente)
	if amb == "" {
		amb = "sandbox"
	}
	if err := h.eredeGatewayRepo.Upsert(idRede, body.PV, body.ClientSecret, amb); err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar")
		return
	}
	wh := h.urlWebhookERede(idRede)
	if amb == "sandbox" && wh != "" {
		if err := erede.RegistrarWebhookSandbox(r.Context(), body.PV, body.ClientSecret, amb, wh, "Bearer", ""); err != nil {
			log.Printf("erede webhook sandbox rede=%s: %v", idRede, err)
		} else {
			log.Printf("erede webhook sandbox registrado rede=%s url=%s", idRede, wh)
		}
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true, "webhook_url": wh})
}

func (h *Handlers) urlWebhookERede(idRede string) string {
	base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + "/v1/public/erede/webhook/" + strings.TrimSpace(idRede)
}

func (h *Handlers) urlWebhookERedePosto(idRede, idPosto string) string {
	base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + "/v1/public/erede/webhook/" + strings.TrimSpace(idRede) + "/" + strings.TrimSpace(idPosto)
}

// truncarLogWebhookERede limita o corpo no log (evita linhas gigantes).
func truncarLogWebhookERede(body []byte) string {
	s := strings.TrimSpace(string(body))
	const max = 512
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
