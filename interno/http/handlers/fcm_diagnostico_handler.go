package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
)

// GetPushDiagnosticoRedePainel GET — contagens FCM da sessão (gestor ou gerente de posto).
func (h *Handlers) GetPushDiagnosticoRedePainel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
	h.escreverDiagnosticoPushRede(w, idRede)
}

// GetPushDiagnosticoRedeAdmin GET — super-admin informa ?id_rede=UUID.
func (h *Handlers) GetPushDiagnosticoRedeAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede na query")
		return
	}
	h.escreverDiagnosticoPushRede(w, idRede)
}

func (h *Handlers) escreverDiagnosticoPushRede(w http.ResponseWriter, idRede string) {
	stats, err := h.usuarioRedeService.DiagnosticoPushRede(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, err.Error())
		return
	}
	fcmOk := strings.TrimSpace(h.cfg.FcmCaminhoContaServico) != ""

	sugestoes := []string{}
	if !fcmOk {
		sugestoes = append(sugestoes, "Defina FCM_SA (JSON da conta de servico Firebase) no servidor e reinicie a API.")
	}
	if stats.TokensDistintos == 0 {
		sugestoes = append(sugestoes, "Nenhum token FCM nesta rede. No app (Android/iOS), login como cliente, permissao de notificacoes e rede online para registar POST /v1/eu/push/fcm.")
	}
	if stats.ClientesAtivos > 0 && stats.ClientesComTokenFCM == 0 {
		sugestoes = append(sugestoes, fmt.Sprintf("Ha %d cliente(s) ativo(s) nesta rede mas nenhum com token FCM na base.", stats.ClientesAtivos))
	}

	envioOK := fcmOk && stats.TokensDistintos > 0

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_rede":                               idRede,
		"fcm_credencial_servidor_configurada": fcmOk,
		"clientes_ativos":                       stats.ClientesAtivos,
		"clientes_com_token_fcm":               stats.ClientesComTokenFCM,
		"tokens_fcm_distintos":                  stats.TokensDistintos,
		"envio_push_campanha_possivel":          envioOK,
		"campanha_envia_push_quando": []string{
			"Status da campanha = ATIVA.",
			"valida_no_app = true.",
			"Criacao ja como ATIVA ou edicao que passa a ATIVA+valida no app (transicao). Alteracoes em campanha ja ATIVA sem mudar estado nao reenviam push.",
		},
		"onde_testar": map[string]string{
			"app_cliente_autenticado":       "POST /v1/eu/push/fcm/teste",
			"painel_gestor_ou_gerente":       "POST /v1/gestor-rede/dev/push/fcm/rede/teste",
			"este_relatorio":                "GET /v1/gestor-rede/dev/push/diagnostico (sessao gestor/gerente) ou GET /v1/admin/redes/dev/push/diagnostico?id_rede=UUID",
			"log_servidor":                  "Linhas prefixadas com \"fcm campanha:\" ao criar/editar campanha.",
		},
		"sugestoes": sugestoes,
	})
}
