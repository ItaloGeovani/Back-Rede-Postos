package handlers

import (
	"errors"
	"net/http"
	"strings"

	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// GireGanheConfigGestor GET/PATCH /v1/gestor-rede/dev/redes/gire-ganhe
func (h *Handlers) GireGanheConfigGestor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getGireGanheConfigGestor(w, r)
	case http.MethodPatch:
		h.patchGireGanheConfigGestor(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) getGireGanheConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.gireGanhe == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	cfg, err := h.gireGanhe.BuscarConfigGestor(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	premios := make([]map[string]any, 0, len(cfg.PremiosEspeciais))
	for _, p := range cfg.PremiosEspeciais {
		premios = append(premios, map[string]any{"valor_moedas": p.ValorMoedas, "percentual": p.Percentual})
	}
	pers := make([]map[string]any, 0, len(cfg.PremiosRoletaPersonalizada))
	for _, p := range cfg.PremiosRoletaPersonalizada {
		pers = append(pers, map[string]any{"valor_moedas": p.ValorMoedas, "percentual": p.Percentual})
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"custo_moedas":                 cfg.CustoMoedas,
		"premio_min_moedas":            cfg.PremioMinMoeda,
		"premio_max_moedas":            cfg.PremioMaxMoeda,
		"giros_max_dia":                cfg.GirosMaxDia,
		"timezone":                     strings.TrimSpace(cfg.Timezone),
		"primeiro_giro_gratis_ativo":   cfg.PrimeiroGiroGratisAtivo,
		"roleta_modo":                  strings.TrimSpace(cfg.RoletaModo),
		"premios_roleta_personalizada": pers,
		"premios_especiais_ativo":      cfg.PremiosEspeciaisAtivo,
		"premios_especiais":            premios,
	})
}

type reqGireGanheConfig struct {
	CustoMoedas                float64 `json:"custo_moedas"`
	PremioMinMoedas            float64 `json:"premio_min_moedas"`
	PremioMaxMoedas            float64 `json:"premio_max_moedas"`
	GirosMaxDia                int     `json:"giros_max_dia"`
	Timezone                   string  `json:"timezone"`
	PrimeiroGiroGratisAtivo    bool    `json:"primeiro_giro_gratis_ativo"`
	PremiosEspeciaisAtivo      bool    `json:"premios_especiais_ativo"`
	RoletaModo                 string  `json:"roleta_modo"`
	PremiosRoletaPersonalizada []struct {
		ValorMoedas float64 `json:"valor_moedas"`
		Percentual  float64 `json:"percentual"`
	} `json:"premios_roleta_personalizada"`
	PremiosEspeciais []struct {
		ValorMoedas float64 `json:"valor_moedas"`
		Percentual  float64 `json:"percentual"`
	} `json:"premios_especiais"`
}

func (h *Handlers) patchGireGanheConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.gireGanhe == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqGireGanheConfig
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	tz := strings.TrimSpace(req.Timezone)
	var premios []repositorios.GireGanhePremioEspecial
	for _, row := range req.PremiosEspeciais {
		premios = append(premios, repositorios.GireGanhePremioEspecial{
			ValorMoedas: row.ValorMoedas,
			Percentual:  row.Percentual,
		})
	}
	var personal []repositorios.GireGanhePremioEspecial
	for _, row := range req.PremiosRoletaPersonalizada {
		personal = append(personal, repositorios.GireGanhePremioEspecial{
			ValorMoedas: row.ValorMoedas,
			Percentual:  row.Percentual,
		})
	}
	modo := strings.TrimSpace(req.RoletaModo)
	if err := h.gireGanhe.SalvarConfigGestor(idRede, req.CustoMoedas, req.PremioMinMoedas, req.PremioMaxMoedas, req.GirosMaxDia, tz, req.PrimeiroGiroGratisAtivo, modo, personal, req.PremiosEspeciaisAtivo, premios); err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, "confira custo, faixa de premio, timezone, giros por dia; modo personalizado: lista valor + % deve somar 100; modo padrao: jackpots com soma <= 100 e valor > premio max quando ativo")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar")
		return
	}
	cfg, _ := h.gireGanhe.BuscarConfigGestor(idRede)
	premiosOut := make([]map[string]any, 0, len(cfg.PremiosEspeciais))
	for _, p := range cfg.PremiosEspeciais {
		premiosOut = append(premiosOut, map[string]any{"valor_moedas": p.ValorMoedas, "percentual": p.Percentual})
	}
	persOut := make([]map[string]any, 0, len(cfg.PremiosRoletaPersonalizada))
	for _, p := range cfg.PremiosRoletaPersonalizada {
		persOut = append(persOut, map[string]any{"valor_moedas": p.ValorMoedas, "percentual": p.Percentual})
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":                     "configuracao do gire e ganhe salva",
		"custo_moedas":                 cfg.CustoMoedas,
		"premio_min_moedas":            cfg.PremioMinMoeda,
		"premio_max_moedas":            cfg.PremioMaxMoeda,
		"giros_max_dia":                cfg.GirosMaxDia,
		"timezone":                     cfg.Timezone,
		"primeiro_giro_gratis_ativo":   cfg.PrimeiroGiroGratisAtivo,
		"roleta_modo":                  strings.TrimSpace(cfg.RoletaModo),
		"premios_roleta_personalizada": persOut,
		"premios_especiais_ativo":      cfg.PremiosEspeciaisAtivo,
		"premios_especiais":            premiosOut,
	})
}
