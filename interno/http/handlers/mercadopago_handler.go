package handlers

import (
	"encoding/json"
	"errors"
	"io"
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

// MercadoPagoWebhookPublico POST /v1/public/mercadopago/webhook/{rede_id} ou .../{rede_id}/{posto_id}
func (h *Handlers) MercadoPagoWebhookPublico(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	const prefix = "/v1/public/mercadopago/webhook/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/")
	parts := strings.Split(rest, "/")
	idRede := ""
	idPosto := ""
	if len(parts) >= 1 {
		idRede = strings.TrimSpace(parts[0])
	}
	if len(parts) >= 2 {
		idPosto = strings.TrimSpace(parts[1])
	}
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "rede_id invalido")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "corpo invalido")
		return
	}
	_ = r.Body.Close()

	var creds *repositorios.MercadoPagoGatewayCredenciais
	if idPosto != "" {
		creds, err = h.mpGatewayRepo.BuscarPorPostoID(idPosto, idRede)
	} else {
		creds, err = h.mpGatewayRepo.BuscarPorRedeID(idRede)
	}
	if err != nil {
		if errors.Is(err, repositorios.ErrMercadoPagoGatewayNaoConfigurado) ||
			errors.Is(err, repositorios.ErrMercadoPagoGatewayPostoNaoConfigurado) {
			log.Printf("mercadopago webhook: rede=%s posto=%s sem credenciais (ignorado)", idRede, idPosto)
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Printf("mercadopago webhook: buscar creds rede=%s posto=%s: %v", idRede, idPosto, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if strings.TrimSpace(creds.WebhookSecret) == "" {
		log.Printf("mercadopago webhook: rede %s sem mp_webhook_secret", idRede)
		w.WriteHeader(http.StatusOK)
		return
	}

	var actionWrap struct {
		Action string `json:"action"`
	}
	_ = json.Unmarshal(body, &actionWrap)
	if actionWrap.Action != "" && !strings.HasPrefix(actionWrap.Action, "payment.") {
		// Ex.: plan, subscription — ignorar
		w.WriteHeader(http.StatusOK)
		return
	}

	dataID, err := servicos.ExtrairDataIDDoWebhookMercadoPago(body)
	if err != nil {
		log.Printf("mercadopago webhook: extrair data.id: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	xSig := r.Header.Get("x-signature")
	xReq := r.Header.Get("x-request-id")
	if !servicos.ValidarAssinaturaWebhookMercadoPago(creds.WebhookSecret, body, xSig, xReq, dataID) {
		log.Printf("mercadopago webhook: assinatura invalida rede=%s", idRede)
		utils.ResponderErro(w, http.StatusForbidden, "assinatura invalida")
		return
	}

	paymentID, err := strconv.Atoi(dataID)
	if err != nil {
		log.Printf("mercadopago webhook: data.id nao numerico: %q", dataID)
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	pay, err := servicos.ConsultarPagamentoMercadoPago(ctx, creds.AccessToken, paymentID)
	if err != nil {
		log.Printf("mercadopago webhook: consultar payment %d: %v", paymentID, err)
		w.WriteHeader(http.StatusOK)
		return
	}
	if strings.TrimSpace(pay.Status) != "approved" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if h.voucherCompraSvc != nil {
		h.voucherCompraSvc.ProcessarPagamentoAprovadoMercadoPago(idRede, strings.TrimSpace(pay.ExternalReference))
	}
	servicos.LogPagamentoAprovadoWebhook(idRede, paymentID, pay.ExternalReference, pay.Status)
	w.WriteHeader(http.StatusOK)
}

// MercadoPagoGatewayGestor GET/PUT — gestor (rede ou postos) ou gerente (posto da sessão, modo POSTO).
func (h *Handlers) MercadoPagoGatewayGestor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getMercadoPagoGatewayPainel(w, r)
	case http.MethodPut:
		h.putMercadoPagoGatewayPainel(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

// MercadoPagoGatewayPostoGestor PUT /v1/gestor-rede/dev/mercadopago-gateway/posto
func (h *Handlers) MercadoPagoGatewayPostoGestor(w http.ResponseWriter, r *http.Request) {
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
	modo := servicos.NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo)
	if modo != modelos.GatewayPagamentoModoPosto {
		utils.ResponderErro(w, http.StatusBadRequest, "rede usa gateway unico; altere o modo em Gateways de pagamento")
		return
	}
	var body struct {
		IDPosto       string `json:"id_posto"`
		AccessToken   string `json:"mp_access_token"`
		WebhookSecret string `json:"mp_webhook_secret"`
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
	if strings.TrimSpace(body.AccessToken) == "" || strings.TrimSpace(body.WebhookSecret) == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "mp_access_token e mp_webhook_secret sao obrigatorios")
		return
	}
	if err := h.mpGatewayRepo.UpsertPosto(idPosto, idRede, body.AccessToken, body.WebhookSecret); err != nil {
		log.Printf("mercadopago upsert posto=%s rede=%s: %v", idPosto, idRede, err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar credenciais do posto")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"webhook_url": h.urlWebhookMercadoPagoPosto(idRede, idPosto),
	})
}

// EditarGatewayPagamentoModoGestor PUT /v1/gestor-rede/dev/redes/gateway-pagamento-modo
func (h *Handlers) EditarGatewayPagamentoModoGestor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var body struct {
		Modo string `json:"gateway_pagamento_modo"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	rede, err := h.redeService.EditarGatewayPagamentoModo(idRede, body.Modo)
	if err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, "modo invalido (use REDE ou POSTO)")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar modo")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":                     true,
		"gateway_pagamento_modo": rede.GatewayPagamentoModo,
	})
}

func (h *Handlers) getMercadoPagoGatewayPainel(w http.ResponseWriter, r *http.Request) {
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
		"gateway_pagamento_modo":    modo,
		"gateway_provedor_ativo":    servicos.NormalizarGatewayProvedorAtivo(rede.GatewayProvedorAtivo),
		"gateway_meios_habilitados": rede.GatewayMeiosHabilitados,
	}

	if u != nil && u.Papel == modelos.PapelGerentePosto {
		idPosto := strings.TrimSpace(u.IDPosto)
		if modo != modelos.GatewayPagamentoModoPosto {
			out["mensagem"] = "Esta rede usa uma conta Mercado Pago unica (configurada pelo gestor)."
			utils.ResponderJSON(w, http.StatusOK, out)
			return
		}
		out["id_posto"] = idPosto
		out["webhook_url"] = h.urlWebhookMercadoPagoPosto(idRede, idPosto)
		creds, errC := h.mpGatewayRepo.BuscarPorPostoID(idPosto, idRede)
		h.preencherCredenciaisMPNoJSON(out, creds, errC)
		utils.ResponderJSON(w, http.StatusOK, out)
		return
	}

	if modo == modelos.GatewayPagamentoModoPosto {
		postos, err := h.mpGatewayRepo.ListarStatusPostosPorRede(idRede)
		if err != nil {
			log.Printf("mercadopago listar postos rede=%s: %v", idRede, err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar postos")
			return
		}
		base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicBaseURL), "/")
		itens := make([]map[string]any, 0, len(postos))
		for _, p := range postos {
			wh := ""
			if base != "" {
				wh = base + "/v1/public/mercadopago/webhook/" + idRede + "/" + p.PostoID
			}
			itens = append(itens, map[string]any{
				"id_posto":                      p.PostoID,
				"nome":                          p.Nome,
				"codigo":                        p.Codigo,
				"webhook_url":                   wh,
				"mp_access_token_configurado":   p.MpAccessTokenOK,
				"mp_webhook_secret_configurado": p.MpWebhookSecretOK,
			})
		}
		out["postos"] = itens
		utils.ResponderJSON(w, http.StatusOK, out)
		return
	}

	out["webhook_url"] = h.urlWebhookMercadoPago(idRede)
	creds, errC := h.mpGatewayRepo.BuscarPorRedeID(idRede)
	h.preencherCredenciaisMPNoJSON(out, creds, errC)
	utils.ResponderJSON(w, http.StatusOK, out)
}

func (h *Handlers) putMercadoPagoGatewayPainel(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	u := middlewares.Usuario(r.Context())
	if u != nil && u.Papel == modelos.PapelGerentePosto {
		utils.ResponderErro(w, http.StatusForbidden, "gerente deve usar salvar credenciais do posto (PUT mercadopago-gateway/posto)")
		return
	}
	rede, err := h.redeService.BuscarPorID(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		return
	}
	if servicos.NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo) != modelos.GatewayPagamentoModoRede {
		utils.ResponderErro(w, http.StatusBadRequest, "no modo por posto use PUT mercadopago-gateway/posto para cada unidade")
		return
	}
	var body struct {
		AccessToken   string `json:"mp_access_token"`
		WebhookSecret string `json:"mp_webhook_secret"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	if strings.TrimSpace(body.AccessToken) == "" || strings.TrimSpace(body.WebhookSecret) == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "mp_access_token e mp_webhook_secret sao obrigatorios")
		return
	}
	if err := h.mpGatewayRepo.Upsert(idRede, body.AccessToken, body.WebhookSecret); err != nil {
		log.Printf("mercadopago upsert rede=%s: %v", idRede, err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar credenciais")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"webhook_url": h.urlWebhookMercadoPago(idRede),
	})
}

func (h *Handlers) preencherCredenciaisMPNoJSON(out map[string]any, creds *repositorios.MercadoPagoGatewayCredenciais, err error) {
	out["mp_access_token_configurado"] = false
	out["mp_webhook_secret_configurado"] = false
	if err != nil {
		return
	}
	configurado := strings.TrimSpace(creds.AccessToken) != ""
	secretOk := strings.TrimSpace(creds.WebhookSecret) != ""
	out["mp_access_token_configurado"] = configurado
	out["mp_webhook_secret_configurado"] = secretOk
	if configurado {
		out["mp_access_token_mascarado"] = mascararSegredoMercadoPago(creds.AccessToken)
	}
	if secretOk {
		out["mp_webhook_secret_mascarado"] = mascararSegredoMercadoPago(creds.WebhookSecret)
	}
}

// PostClienteMercadoPagoPix POST /v1/eu/pagamentos/mercadopago/pix — cliente autenticado; valor validado no servidor.
func (h *Handlers) PostClienteMercadoPagoPix(w http.ResponseWriter, r *http.Request) {
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
		utils.ResponderErro(w, http.StatusForbidden, "disponivel apenas para cliente")
		return
	}

	var body struct {
		Valor             float64 `json:"valor"`
		Descricao         string  `json:"descricao"`
		PayerEmail        string  `json:"payer_email"`
		DocTipo           string  `json:"doc_tipo"`
		DocNumero         string  `json:"doc_numero"`
		ExternalReference string  `json:"external_reference"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "json invalido")
		return
	}
	if body.Valor < 1.0 {
		utils.ResponderErro(w, http.StatusBadRequest, "valor minimo R$ 1,00")
		return
	}
	body.PayerEmail = strings.TrimSpace(body.PayerEmail)
	if body.PayerEmail == "" || !strings.Contains(body.PayerEmail, "@") {
		utils.ResponderErro(w, http.StatusBadRequest, "payer_email invalido")
		return
	}
	body.DocTipo = strings.TrimSpace(body.DocTipo)
	body.DocNumero = strings.TrimSpace(body.DocNumero)
	if body.DocTipo == "" || body.DocNumero == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "doc_tipo e doc_numero obrigatorios (ex: CPF e 11 digitos)")
		return
	}

	creds, err := h.mpGatewayRepo.BuscarPorRedeID(u.IDRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrMercadoPagoGatewayNaoConfigurado) {
			utils.ResponderErro(w, http.StatusBadRequest, "rede sem mercado pago configurado")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar gateway")
		return
	}
	if strings.TrimSpace(creds.AccessToken) == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "rede sem mp_access_token")
		return
	}

	base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicBaseURL), "/")
	if base == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "servidor sem PUBLIC_BASE_URL: necessario para notification_url do PIX")
		return
	}
	notif := base + "/v1/public/mercadopago/webhook/" + strings.TrimSpace(u.IDRede)

	desc := strings.TrimSpace(body.Descricao)
	if desc == "" {
		desc = "Pagamento AP GasPass"
	}
	ext := strings.TrimSpace(body.ExternalReference)
	if ext == "" {
		ext = "rede:" + u.IDRede + ";usuario:" + u.IDUsuario
	}

	ctx := r.Context()
	res, err := servicos.CriarCobrancaPixMercadoPago(ctx, creds.AccessToken, servicos.CriarCobrancaPixMercadoPagoInput{
		Valor:             body.Valor,
		Descricao:         desc,
		PayerEmail:        body.PayerEmail,
		DocTipo:           body.DocTipo,
		DocNumero:         body.DocNumero,
		ExternalReference: ext,
		NotificationURL:   notif,
	})
	if err != nil {
		log.Printf("mercadopago criar pix rede=%s: %v", u.IDRede, err)
		utils.ResponderErro(w, http.StatusBadGateway, "falha ao criar cobranca no mercado pago")
		return
	}

	qr := res.PointOfInteraction.TransactionData.QRCode
	qrB64 := res.PointOfInteraction.TransactionData.QRCodeBase64
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"payment_id":      res.ID,
		"status":          res.Status,
		"status_detail":   res.StatusDetail,
		"qr_code":         qr,
		"qr_code_base64":  qrB64,
		"amount":          body.Valor,
		"external_reference": res.ExternalReference,
	})
}

func (h *Handlers) urlWebhookMercadoPago(idRede string) string {
	base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + "/v1/public/mercadopago/webhook/" + strings.TrimSpace(idRede)
}

func (h *Handlers) urlWebhookMercadoPagoPosto(idRede, idPosto string) string {
	base := strings.TrimRight(strings.TrimSpace(h.cfg.PublicBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + "/v1/public/mercadopago/webhook/" + strings.TrimSpace(idRede) + "/" + strings.TrimSpace(idPosto)
}

func mascararSegredoMercadoPago(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= 12 {
		return "****"
	}
	return s[:6] + "…" + s[len(s)-4:]
}
