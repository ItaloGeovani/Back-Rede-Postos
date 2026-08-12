package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
)

type revogarSessoesClientesBody struct {
	IDRede          string `json:"id_rede"`
	LimparTokensFCM *bool  `json:"limpar_tokens_fcm"`
}

// PostRevogarSessoesClientesAdmin POST /v1/admin/sessoes/revogar-clientes
// Invalida sessões tok_* dos clientes do app (força novo login). Por padrão também limpa tokens FCM.
func (h *Handlers) PostRevogarSessoesClientesAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "use POST")
		return
	}

	var body revogarSessoesClientesBody
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	idRede := strings.TrimSpace(body.IDRede)
	limparFCM := true
	if body.LimparTokensFCM != nil {
		limparFCM = *body.LimparTokensFCM
	}

	sessoes, err := h.autenticador.RevogarSessoesPorPapel(modelos.PapelCliente, idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, err.Error())
		return
	}

	var tokensFCM int64
	if limparFCM && h.usuarioRedeService != nil {
		tokensFCM, err = h.usuarioRedeService.RemoverTokensFCMClientes(idRede)
		if err != nil {
			utils.ResponderErro(w, http.StatusInternalServerError, "sessoes revogadas, mas falha ao limpar FCM: "+err.Error())
			return
		}
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":             "clientes serao deslogados na proxima chamada autenticada",
		"papel":                string(modelos.PapelCliente),
		"id_rede":              idRede,
		"sessoes_revogadas":    sessoes,
		"tokens_fcm_removidos": tokensFCM,
		"limpar_tokens_fcm":    limparFCM,
	})
}
