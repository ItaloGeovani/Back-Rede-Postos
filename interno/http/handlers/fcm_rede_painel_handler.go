package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/notificacoes"
	"gaspass-servidor/utils"
)

// PostFcmTesteRedePainel POST /v1/.../push/fcm/rede/teste — envia teste a todos os app clientes (tokens) da rede.
// Gestor da rede e gerente de posto: JWT com rede; titulo e corpo opcionais.
func (h *Handlers) PostFcmTesteRedePainel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "nao autenticado")
		return
	}
	if u.Papel != modelos.PapelGestorRede && u.Papel != modelos.PapelGerentePosto {
		utils.ResponderErro(w, http.StatusForbidden, "acesso negado")
		return
	}
	idRede := strings.TrimSpace(u.IDRede)
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "usuario sem rede vinculada")
		return
	}

	var body struct {
		Titulo string `json:"titulo"`
		Corpo  string `json:"corpo"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	titulo := strings.TrimSpace(body.Titulo)
	corpo := strings.TrimSpace(body.Corpo)

	cred := strings.TrimSpace(h.cfg.FcmCaminhoContaServico)
	projectID := notificacoes.ProjectIDDaCredencial(cred)
	log.Printf("fcm teste rede: inicio papel=%s id_rede=%s titulo=%q corpo=%q fcm_sa=%q project_id=%q",
		u.Papel, idRede, titulo, corpo, cred, projectID)
	if cred == "" {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "push nao configurado no servidor (defina FCM_SA)")
		return
	}
	tokens, err := h.usuarioRedeService.ListarTokensFCMClientesRede(idRede)
	if err != nil {
		log.Printf("fcm teste rede: listar tokens: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar tokens da rede")
		return
	}
	log.Printf("fcm teste rede: tokens encontrados=%d", len(tokens))
	if len(tokens) == 0 {
		utils.ResponderErro(w, http.StatusBadRequest, "nenhum token FCM de clientes nesta rede. E preciso abrir o app (cliente) e permitir notificacoes.")
		return
	}
	xctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	ok, fal, invalidos, err := notificacoes.EnviarTesteRede(xctx, cred, tokens, idRede, titulo, corpo)
	if err != nil {
		log.Printf("fcm teste rede: envio falhou: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, err.Error())
		return
	}
	removidos := int64(0)
	if len(invalidos) > 0 {
		removidos, err = h.usuarioRedeService.RemoverTokensFCM(invalidos)
		if err != nil {
			log.Printf("fcm teste rede: limpar tokens: %v", err)
		} else {
			log.Printf("fcm teste rede: removidos %d token(s) invalidos da base", removidos)
		}
	}
	log.Printf("fcm teste rede: fim enviados=%d falhas=%d invalidos=%d removidos=%d project_id=%s",
		ok, fal, len(invalidos), removidos, projectID)
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"enviados":         ok,
		"falhas":           fal,
		"tokens_tentado":   len(tokens),
		"tokens_invalidos": len(invalidos),
		"tokens_removidos": removidos,
		"fcm_project_id":   projectID,
		"fcm_sa_path":      cred,
		"id_rede":          idRede,
		"titulo_enviado":   firstNonEmpty(titulo, "Teste de notificacao"),
		"corpo_enviado":    firstNonEmpty(corpo, "Mensagem de teste do painel."),
		"dica":             "Se falhas=SenderId mismatch: o token do app e de outro projeto Firebase que o FCM_SA do servidor. Lucena espera project_id gasspass-ce536. Reinstale o app apos alinhar o google-services.json e o FCM_SA.",
	})
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}
