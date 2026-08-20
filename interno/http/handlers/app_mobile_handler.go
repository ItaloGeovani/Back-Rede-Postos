package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
)

func (h *Handlers) VerificarVersaoAppMobile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	q := r.URL.Query()
	plataforma := strings.ToLower(strings.TrimSpace(q.Get("plataforma")))
	instalada := strings.TrimSpace(q.Get("versao_instalada"))
	if instalada == "" {
		instalada = strings.TrimSpace(q.Get("versao"))
	}
	idRedeQ := strings.TrimSpace(q.Get("id_rede"))
	if plataforma != "ios" && plataforma != "android" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe plataforma=ios ou plataforma=android")
		return
	}
	if instalada == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe versao_instalada (ex.: 1.0.0)")
		return
	}

	var cfg *modelos.ConfiguracaoAppMobile
	var err error
	if idRedeQ != "" {
		if _, e := uuid.Parse(idRedeQ); e != nil {
			utils.ResponderErro(w, http.StatusBadRequest, "id_rede invalido")
			return
		}
		var ok bool
		cfg, ok, err = h.appMobileRedeRepo.Obter(idRedeQ)
		if err != nil {
			log.Printf("verificar versao app mobile rede: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao da rede")
			return
		}
		if !ok {
			cfg, err = h.appMobileRepo.Obter()
		}
	} else {
		cfg, err = h.appMobileRepo.Obter()
	}
	if err != nil {
		log.Printf("verificar versao app mobile: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	if cfg == nil {
		cfg = &modelos.ConfiguracaoAppMobile{VersaoIOS: "0.0.0", VersaoAndroid: "0.0.0"}
	}

	var atual string
	var urlLoja string
	if plataforma == "ios" {
		atual = cfg.VersaoIOS
		urlLoja = cfg.URLLojaIOS
	} else {
		atual = cfg.VersaoAndroid
		urlLoja = cfg.URLLojaAndroid
	}
	if strings.TrimSpace(atual) == "" {
		atual = "0.0.0"
	}

	desatualizada := utils.VersaoSemverMenor(instalada, atual)
	atualizacaoDisponivel := desatualizada

	out := map[string]any{
		"plataforma":                 plataforma,
		"versao_instalada":           instalada,
		"versao_atual_servidor":      atual,
		"atualizacao_disponivel":     atualizacaoDisponivel,
		"instalada_desatualizada":    desatualizada,
		"url_loja":                   urlLoja,
		"mensagem":                   cfg.MensagemAtualizacao,
		"atualizacao_obrigatoria":    cfg.AtualizacaoObrigatoria,
		"deve_exibir_modal_atualizar": atualizacaoDisponivel,
	}
	if idRedeQ != "" {
		out["id_rede"] = idRedeQ
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

// PublicAppLojas GET /v1/public/app-lojas?id_rede= — URLs das lojas (iOS e Android) para landing.
func (h *Handlers) PublicAppLojas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRedeQ := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	if idRedeQ == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	if _, e := uuid.Parse(idRedeQ); e != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "id_rede invalido")
		return
	}

	var cfg *modelos.ConfiguracaoAppMobile
	var err error
	cfg, ok, err := h.appMobileRedeRepo.Obter(idRedeQ)
	if err != nil {
		log.Printf("public app-lojas rede: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	if !ok || cfg == nil {
		cfg, err = h.appMobileRepo.Obter()
		if err != nil {
			log.Printf("public app-lojas global: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
			return
		}
	}
	if cfg == nil {
		cfg = &modelos.ConfiguracaoAppMobile{}
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_rede":          idRedeQ,
		"url_loja_ios":     strings.TrimSpace(cfg.URLLojaIOS),
		"url_loja_android": strings.TrimSpace(cfg.URLLojaAndroid),
	})
}

// AppMobileVersaoAdmin GET le / PUT salva configuracao dos apps (super admin).
func (h *Handlers) AppMobileVersaoAdmin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.obterConfigAppMobileAdmin(w, r)
	case http.MethodPut, http.MethodPatch:
		h.salvarConfigAppMobileAdmin(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) obterConfigAppMobileAdmin(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.appMobileRepo.Obter()
	if err != nil {
		log.Printf("obter config app mobile admin: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"configuracao": cfg,
	})
}

type reqSalvarAppMobile struct {
	VersaoIOS              string `json:"versao_ios"`
	VersaoAndroid          string `json:"versao_android"`
	URLLojaIOS             string `json:"url_loja_ios"`
	URLLojaAndroid         string `json:"url_loja_android"`
	MensagemAtualizacao    string `json:"mensagem_atualizacao"`
	AtualizacaoObrigatoria *bool  `json:"atualizacao_obrigatoria"`
}

func (h *Handlers) salvarConfigAppMobileAdmin(w http.ResponseWriter, r *http.Request) {
	var req reqSalvarAppMobile
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	atual, err := h.appMobileRepo.Obter()
	if err != nil {
		log.Printf("salvar config app mobile obter: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao atual")
		return
	}
	out := &modelos.ConfiguracaoAppMobile{
		VersaoIOS:              strings.TrimSpace(req.VersaoIOS),
		VersaoAndroid:          strings.TrimSpace(req.VersaoAndroid),
		URLLojaIOS:             strings.TrimSpace(req.URLLojaIOS),
		URLLojaAndroid:         strings.TrimSpace(req.URLLojaAndroid),
		MensagemAtualizacao:    strings.TrimSpace(req.MensagemAtualizacao),
		AtualizacaoObrigatoria: atual.AtualizacaoObrigatoria,
	}
	if req.AtualizacaoObrigatoria != nil {
		out.AtualizacaoObrigatoria = *req.AtualizacaoObrigatoria
	}
	if out.VersaoIOS == "" {
		out.VersaoIOS = "0.0.0"
	}
	if out.VersaoAndroid == "" {
		out.VersaoAndroid = "0.0.0"
	}
	if err := h.appMobileRepo.Salvar(out); err != nil {
		log.Printf("salvar config app mobile: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar configuracao")
		return
	}
	salvo, err := h.appMobileRepo.Obter()
	if err != nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{"mensagem": "configuracao salva"})
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":       "configuracao salva",
		"configuracao": salvo,
	})
}

// AppMobileVersaoRedeAdmin GET/PUT: versoes do app de cliente **por rede** (super admin).
// GET: ?id_rede=UUID devolve { configuracao_rede, possui_sobrescritura, configuracao_global }.
// PUT: corpo com id_rede e mesmos campos de reqSalvarAppMobile.
func (h *Handlers) AppMobileVersaoRedeAdmin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.obterAppMobileVersaoRedeAdmin(w, r)
	case http.MethodPut, http.MethodPatch:
		h.salvarAppMobileVersaoRedeAdmin(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) obterAppMobileVersaoRedeAdmin(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	idRede := strings.TrimSpace(q.Get("id_rede"))
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	if _, e := uuid.Parse(idRede); e != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "id_rede invalido")
		return
	}
	global, err := h.appMobileRepo.Obter()
	if err != nil {
		log.Printf("app mobile rede admin global: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao global")
		return
	}
	rede, ok, err := h.appMobileRedeRepo.Obter(idRede)
	if err != nil {
		log.Printf("app mobile rede admin: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao da rede")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_rede":                 idRede,
		"configuracao_rede":       rede,
		"possui_sobrescritura":     ok,
		"configuracao_global":     global,
	})
}

type reqSalvarAppMobileRede struct {
	IDRede                 string `json:"id_rede"`
	VersaoIOS              string `json:"versao_ios"`
	VersaoAndroid          string `json:"versao_android"`
	URLLojaIOS             string `json:"url_loja_ios"`
	URLLojaAndroid         string `json:"url_loja_android"`
	MensagemAtualizacao    string `json:"mensagem_atualizacao"`
	AtualizacaoObrigatoria *bool  `json:"atualizacao_obrigatoria"`
}

func (h *Handlers) salvarAppMobileVersaoRedeAdmin(w http.ResponseWriter, r *http.Request) {
	var req reqSalvarAppMobileRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	idRede := strings.TrimSpace(req.IDRede)
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	if _, e := uuid.Parse(idRede); e != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "id_rede invalido")
		return
	}
	atualGlobal, err := h.appMobileRepo.Obter()
	if err != nil {
		log.Printf("app mobile rede admin obter global: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao global")
		return
	}
	out := &modelos.ConfiguracaoAppMobile{
		VersaoIOS:              strings.TrimSpace(req.VersaoIOS),
		VersaoAndroid:          strings.TrimSpace(req.VersaoAndroid),
		URLLojaIOS:             strings.TrimSpace(req.URLLojaIOS),
		URLLojaAndroid:         strings.TrimSpace(req.URLLojaAndroid),
		MensagemAtualizacao:    strings.TrimSpace(req.MensagemAtualizacao),
		AtualizacaoObrigatoria: atualGlobal.AtualizacaoObrigatoria,
	}
	if req.AtualizacaoObrigatoria != nil {
		out.AtualizacaoObrigatoria = *req.AtualizacaoObrigatoria
	}
	if out.VersaoIOS == "" {
		out.VersaoIOS = "0.0.0"
	}
	if out.VersaoAndroid == "" {
		out.VersaoAndroid = "0.0.0"
	}
	if err := h.appMobileRedeRepo.Salvar(idRede, out); err != nil {
		log.Printf("app mobile salvar rede: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar configuracao")
		return
	}
	rede, ok, err := h.appMobileRedeRepo.Obter(idRede)
	if err != nil || !ok {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{
			"mensagem": "configuracao salva",
		})
		return
	}
	global, _ := h.appMobileRepo.Obter()
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":              "configuracao salva",
		"configuracao_rede":     rede,
		"possui_sobrescritura":  true,
		"configuracao_global":   global,
	})
}
