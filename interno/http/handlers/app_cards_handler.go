package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type reqSalvarAppCards struct {
	Cards []struct {
		Slot      int    `json:"slot"`
		Titulo    string `json:"titulo"`
		ImagemURL string `json:"imagem_url"`
		LinkURL   string `json:"link_url"`
		Ativo     bool   `json:"ativo"`
	} `json:"cards"`
}

// AppCardsGestorRede GET lista / PUT salva cards do app para a rede da sessao (gestor ou gerente).
func (h *Handlers) AppCardsGestorRede(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listarAppCardsRede(w, r)
	case http.MethodPut, http.MethodPatch:
		h.salvarAppCardsRede(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) listarAppCardsRede(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	itens, err := h.appCardsRepo.ListarPorRedeID(idRede)
	if err != nil {
		log.Printf("listar app cards: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar cards")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"cards": h.enriquecerRespostaAppCards(itens),
	})
}

func (h *Handlers) salvarAppCardsRede(w http.ResponseWriter, r *http.Request) {
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqSalvarAppCards
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	vistos := make(map[int]bool)
	var lista []*modelos.AppCardRede
	for _, c := range req.Cards {
		if vistos[c.Slot] {
			utils.ResponderErro(w, http.StatusBadRequest, "slot duplicado no payload")
			return
		}
		vistos[c.Slot] = true
		if c.Slot < 0 || c.Slot > 3 {
			utils.ResponderErro(w, http.StatusBadRequest, "slot deve ser 0 (destaque) ou 1 a 3 (promocoes)")
			return
		}
		img := strings.TrimSpace(c.ImagemURL)
		if img != "" && !strings.HasPrefix(strings.ToLower(img), "http://") && !strings.HasPrefix(strings.ToLower(img), "https://") {
			utils.ResponderErro(w, http.StatusBadRequest, "imagem_url deve ser http ou https")
			return
		}
		link := strings.TrimSpace(c.LinkURL)
		if link != "" && !strings.HasPrefix(strings.ToLower(link), "http://") && !strings.HasPrefix(strings.ToLower(link), "https://") {
			utils.ResponderErro(w, http.StatusBadRequest, "link_url deve ser http ou https")
			return
		}
		lista = append(lista, &modelos.AppCardRede{
			Slot:      c.Slot,
			Titulo:    strings.TrimSpace(c.Titulo),
			ImagemURL: img,
			LinkURL:   link,
			Ativo:     c.Ativo,
		})
	}
	if err := h.appCardsRepo.SubstituirPorRede(idRede, lista); err != nil {
		if strings.Contains(err.Error(), "slot invalido") {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("salvar app cards: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar cards")
		return
	}
	itens, err := h.appCardsRepo.ListarPorRedeID(idRede)
	if err != nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{"mensagem": "cards salvos"})
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "cards salvos",
		"cards":    h.enriquecerRespostaAppCards(itens),
	})
}

func (h *Handlers) enriquecerRespostaAppCards(itens []*modelos.AppCardRede) map[string]any {
	var destaque *modelos.AppCardRede
	slots := make([]*modelos.AppCardRede, 3)
	for _, c := range itens {
		if c.Slot == 0 {
			destaque = c
		} else if c.Slot >= 1 && c.Slot <= 3 {
			slots[c.Slot-1] = c
		}
	}
	promos := make([]*modelos.AppCardRede, 0, 3)
	for _, p := range slots {
		if p != nil {
			promos = append(promos, p)
		}
	}
	return map[string]any{
		"lista":          itens,
		"destaque_rede":  destaque,
		"promocoes":      promos,
		"slots_ocupados": len(itens),
	}
}

// PublicListarAppCardsRede GET /v1/public/rede-cards?id_rede=uuid — para o app cliente (sem auth).
func (h *Handlers) PublicListarAppCardsRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	_, err := h.redeService.BuscarPorID(idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrRedeNaoEncontrada) {
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao validar rede")
		return
	}
	itens, err := h.appCardsRepo.ListarPorRedeID(idRede)
	if err != nil {
		log.Printf("public listar app cards: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar cards")
		return
	}
	visiveis := filtrarCardsPublicos(itens)
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_rede": idRede,
		"cards":   h.enriquecerRespostaAppCards(visiveis),
	})
}

func filtrarCardsPublicos(itens []*modelos.AppCardRede) []*modelos.AppCardRede {
	out := make([]*modelos.AppCardRede, 0, len(itens))
	for _, c := range itens {
		if c == nil || !c.Ativo {
			continue
		}
		if strings.TrimSpace(c.ImagemURL) == "" {
			continue
		}
		out = append(out, c)
	}
	return out
}

// PublicRedeInfo GET /v1/public/rede-info?id_rede=uuid — nome fantasia e moeda virtual (app cliente, sem auth).
func (h *Handlers) PublicRedeInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	rede, err := h.redeService.BuscarPorID(idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrRedeNaoEncontrada) {
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
			return
		}
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		return
	}
	if !rede.Ativa {
		utils.ResponderErro(w, http.StatusNotFound, "rede indisponivel")
		return
	}
	// CDNs e proxies costumam cachear GET; flags mudam no painel e o app precisa do valor atual.
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	out := map[string]any{
		"id_rede":                   idRede,
		"nome_fantasia":             strings.TrimSpace(rede.NomeFantasia),
		"telefone":                  strings.TrimSpace(rede.Telefone),
		"moeda_virtual_nome":        strings.TrimSpace(rede.MoedaVirtualNome),
		"moeda_virtual_cotacao":     rede.MoedaVirtualCotacao,
		"app_modulo_indique_ganhe":  rede.AppModuloIndiqueGanhe,
		"app_modulo_checkin_diario": rede.AppModuloCheckinDiario,
		"app_modulo_gire_ganhe":     rede.AppModuloGireGanhe,
		"app_modulo_redes_sociais":  rede.AppModuloRedesSociais,
		"gateway_pagamento_modo":    servicos.NormalizarGatewayPagamentoModo(rede.GatewayPagamentoModo),
		"rede_logo_url":             h.resolveRedeLogoURLPublic(idRede),
	}
	if h.niveisCliente != nil {
		nc, err := h.niveisCliente.Buscar(idRede)
		if err == nil && nc != nil {
			out["app_niveis_moeda_ativo"] = nc.Ativo
			out["app_niveis_mult_desconto_ativo"] = nc.MultDescontoAtivo
			if nc.Ativo {
				out["niveis_cliente"] = nc.Niveis
			} else {
				out["niveis_cliente"] = nil
			}
		} else {
			out["app_niveis_moeda_ativo"] = false
			out["app_niveis_mult_desconto_ativo"] = false
			out["niveis_cliente"] = nil
		}
	} else {
		out["app_niveis_moeda_ativo"] = false
		out["app_niveis_mult_desconto_ativo"] = false
		out["niveis_cliente"] = nil
	}
	if rede.AppModuloCheckinDiario && h.checkinDiario != nil {
		for k, v := range h.checkinDiario.ConfigPublicaParaRede(idRede) {
			out[k] = v
		}
	}
	if rede.AppModuloGireGanhe && h.gireGanhe != nil {
		for k, v := range h.gireGanhe.ConfigPublicaParaRede(idRede) {
			out[k] = v
		}
	}
	if rede.AppModuloRedesSociais && h.redesSociaisRepo != nil {
		links, err := h.redesSociaisRepo.ListarPorRedeID(idRede)
		if err != nil {
			log.Printf("public rede-info redes sociais: %v", err)
			out["redes_sociais"] = []any{}
		} else {
			out["redes_sociais"] = links
		}
	} else {
		out["redes_sociais"] = []any{}
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

// resolveRedeLogoURLPublic URL da marca para o app: imagem do card destaque (slot 0) ativo;
// se vazio, primeiro posto da rede com logo_url preenchido.
func (h *Handlers) resolveRedeLogoURLPublic(idRede string) string {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return ""
	}
	if h.appCardsRepo != nil {
		cards, err := h.appCardsRepo.ListarPorRedeID(idRede)
		if err == nil {
			for _, c := range cards {
				if c != nil && c.Slot == 0 && c.Ativo && strings.TrimSpace(c.ImagemURL) != "" {
					return strings.TrimSpace(c.ImagemURL)
				}
			}
		}
	}
	if h.postoService != nil {
		postos, err := h.postoService.ListarPorRedeID(idRede)
		if err == nil {
			for _, p := range postos {
				if p != nil && strings.TrimSpace(p.LogoURL) != "" {
					return strings.TrimSpace(p.LogoURL)
				}
			}
		}
	}
	return ""
}
