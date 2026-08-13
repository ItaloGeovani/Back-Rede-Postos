package handlers

import (
	"errors"
	"net/http"
	"strings"

	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// CheckinDiarioConfigGestor GET/PATCH /v1/gestor-rede/dev/redes/checkin-diario
func (h *Handlers) CheckinDiarioConfigGestor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getCheckinDiarioConfigGestor(w, r)
	case http.MethodPatch:
		h.patchCheckinDiarioConfigGestor(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) getCheckinDiarioConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.checkinDiario == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	cfg, err := h.checkinDiario.BuscarConfigGestor(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"moedas_por_dia":  cfg.MoedasPorDia,
		"hora_abertura":   strings.TrimSpace(cfg.HoraAbertura),
		"timezone":        strings.TrimSpace(cfg.Timezone),
	})
}

type reqCheckinDiarioConfig struct {
	MoedasPorDia float64 `json:"moedas_por_dia"`
	HoraAbertura string  `json:"hora_abertura"`
	Timezone     string  `json:"timezone"`
}

func (h *Handlers) patchCheckinDiarioConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.checkinDiario == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqCheckinDiarioConfig
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	hz := strings.TrimSpace(req.HoraAbertura)
	if hz == "" {
		hz = "12:00"
	}
	tz := strings.TrimSpace(req.Timezone)
	if err := h.checkinDiario.SalvarConfigGestor(idRede, req.MoedasPorDia, hz, tz); err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, "moedas > 0, hora HH:MM e timezone valido (ex. America/Sao_Paulo)")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar")
		return
	}
	cfg, _ := h.checkinDiario.BuscarConfigGestor(idRede)
	if cfg == nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{"mensagem": "configuracao do check-in diario salva"})
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":       "configuracao do check-in diario salva",
		"moedas_por_dia": cfg.MoedasPorDia,
		"hora_abertura":  cfg.HoraAbertura,
		"timezone":       cfg.Timezone,
	})
}
