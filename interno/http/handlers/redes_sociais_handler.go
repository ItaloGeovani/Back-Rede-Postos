package handlers

import (
	"log"
	"net/http"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
)

var plataformasLinkSocialPermitidas = map[string]struct{}{
	"instagram": {}, "facebook": {}, "youtube": {}, "tiktok": {},
	"whatsapp": {}, "x": {}, "site": {}, "outro": {},
}

type reqSalvarRedesSociais struct {
	Links []struct {
		Plataforma      string `json:"plataforma"`
		TituloExibicao string `json:"titulo_exibicao"`
		URL            string `json:"url"`
	} `json:"links"`
}

// RedesSociaisGestor GET/PATCH /v1/gestor-rede/dev/redes/redes-sociais
func (h *Handlers) RedesSociaisGestor(w http.ResponseWriter, r *http.Request) {
	if h.redesSociaisRepo == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "repositorio indisponivel")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.redesSociaisGestorGet(w, r)
	case http.MethodPut, http.MethodPatch:
		h.redesSociaisGestorPatch(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) redesSociaisGestorGet(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	itens, err := h.redesSociaisRepo.ListarPorRedeID(idRede)
	if err != nil {
		log.Printf("listar redes sociais: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"links": itens})
}

func (h *Handlers) redesSociaisGestorPatch(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqSalvarRedesSociais
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	if len(req.Links) > 20 {
		utils.ResponderErro(w, http.StatusBadRequest, "no maximo 20 links")
		return
	}
	var lista []modelos.RedeLinkSocial
	for _, row := range req.Links {
		p := strings.ToLower(strings.TrimSpace(row.Plataforma))
		if p == "" {
			continue
		}
		if _, ok := plataformasLinkSocialPermitidas[p]; !ok {
			utils.ResponderErro(w, http.StatusBadRequest, "plataforma invalida: "+p)
			return
		}
		tit := strings.TrimSpace(row.TituloExibicao)
		if len(tit) > 80 {
			utils.ResponderErro(w, http.StatusBadRequest, "titulo muito longo (max 80)")
			return
		}
		u := strings.TrimSpace(row.URL)
		if u == "" {
			continue
		}
		low := strings.ToLower(u)
		if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
			utils.ResponderErro(w, http.StatusBadRequest, "url deve comecar com http ou https")
			return
		}
		if len(u) > 512 {
			utils.ResponderErro(w, http.StatusBadRequest, "url muito longa")
			return
		}
		lista = append(lista, modelos.RedeLinkSocial{
			Plataforma:      p,
			TituloExibicao: tit,
			URL:            u,
		})
	}
	if err := h.redesSociaisRepo.Substituir(idRede, lista); err != nil {
		log.Printf("salvar redes sociais: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar")
		return
	}
	itens, err := h.redesSociaisRepo.ListarPorRedeID(idRede)
	if err != nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{"mensagem": "salvo", "links": lista})
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"mensagem": "salvo", "links": itens})
}
