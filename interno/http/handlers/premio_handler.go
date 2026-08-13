package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type reqCriarPremio struct {
	IDRede               string  `json:"id_rede"`
	Titulo               string  `json:"titulo"`
	ImagemURL            string  `json:"imagem_url"`
	ValorMoeda           float64 `json:"valor_moeda"`
	Ativo                *bool   `json:"ativo"`
	VigenciaInicio       *string `json:"vigencia_inicio"`
	VigenciaFim          *string `json:"vigencia_fim"`
	QuantidadeDisponivel *int    `json:"quantidade_disponivel"`
}

type reqEditarPremio struct {
	ID                   string  `json:"id"`
	IDRede               string  `json:"id_rede"`
	Titulo               string  `json:"titulo"`
	ImagemURL            string  `json:"imagem_url"`
	ValorMoeda           float64 `json:"valor_moeda"`
	Ativo                *bool   `json:"ativo"`
	VigenciaInicio       *string `json:"vigencia_inicio"`
	VigenciaFim          *string `json:"vigencia_fim"`
	QuantidadeDisponivel *int    `json:"quantidade_disponivel"`
}

func parseVigenciasPremio(ini, fim *string) (*time.Time, *time.Time, error) {
	if ini == nil || strings.TrimSpace(*ini) == "" {
		return nil, nil, errors.New("vigencia_inicio obrigatoria")
	}
	tIni, err := time.Parse(time.RFC3339, strings.TrimSpace(*ini))
	if err != nil {
		return nil, nil, err
	}
	tIniPtr := &tIni
	var tFimPtr *time.Time
	if fim != nil && strings.TrimSpace(*fim) != "" {
		tFim, err := time.Parse(time.RFC3339, strings.TrimSpace(*fim))
		if err != nil {
			return nil, nil, err
		}
		if tFim.Before(tIni) {
			return nil, nil, errors.New("vigencia_fim antes de vigencia_inicio")
		}
		tFimPtr = &tFim
	}
	return tIniPtr, tFimPtr, nil
}

func boolOuPadraoPremio(v *bool, padrao bool) bool {
	if v == nil {
		return padrao
	}
	return *v
}

func (h *Handlers) ListarPremiosRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	itens, err := h.premioService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede valido")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("listar premios: %v", err)
			utils.ResponderJSON(w, http.StatusInternalServerError, map[string]string{
				"erro":    "falha ao listar premios",
				"detalhe": err.Error(),
			})
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens": itens,
		"total": len(itens),
	})
}

func (h *Handlers) CriarPremioRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	var req reqCriarPremio
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || strings.TrimSpace(u.IDUsuario) == "" {
		utils.ResponderErro(w, http.StatusUnauthorized, "sessao invalida")
		return
	}
	vi, vf, err := parseVigenciasPremio(req.VigenciaInicio, req.VigenciaFim)
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "vigencia_inicio e opcionalmente vigencia_fim em ISO8601 (RFC3339)")
		return
	}
	p, err := h.premioService.Criar(servicos.CriarPremioInput{
		IDRede:               req.IDRede,
		Titulo:               req.Titulo,
		ImagemURL:            req.ImagemURL,
		ValorMoeda:           req.ValorMoeda,
		Ativo:                boolOuPadraoPremio(req.Ativo, true),
		VigenciaInicio:       vi,
		VigenciaFim:          vf,
		QuantidadeDisponivel: req.QuantidadeDisponivel,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos: titulo, valor_moeda, vigencias ou imagem_url")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("criar premio: %v", err)
			utils.ResponderJSON(w, http.StatusInternalServerError, map[string]string{
				"erro":    "falha ao criar premio",
				"detalhe": err.Error(),
			})
		}
		return
	}
	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "premio criado com sucesso",
		"premio":   p,
	})
}

func (h *Handlers) EditarPremioRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	var req reqEditarPremio
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	vi, vf, err := parseVigenciasPremio(req.VigenciaInicio, req.VigenciaFim)
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "vigencia_inicio e opcionalmente vigencia_fim em ISO8601 (RFC3339)")
		return
	}
	err = h.premioService.Atualizar(servicos.AtualizarPremioInput{
		ID:                   req.ID,
		IDRede:               req.IDRede,
		Titulo:               req.Titulo,
		ImagemURL:            req.ImagemURL,
		ValorMoeda:           req.ValorMoeda,
		Ativo:                boolOuPadraoPremio(req.Ativo, true),
		VigenciaInicio:       vi,
		VigenciaFim:          vf,
		QuantidadeDisponivel: req.QuantidadeDisponivel,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case errors.Is(err, repositorios.ErrPremioNaoEncontrado):
			utils.ResponderErro(w, http.StatusNotFound, "premio nao encontrado")
		default:
			log.Printf("editar premio: %v", err)
			utils.ResponderJSON(w, http.StatusInternalServerError, map[string]string{
				"erro":    "falha ao atualizar premio",
				"detalhe": err.Error(),
			})
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "premio atualizado com sucesso",
	})
}

// PublicListarPremios GET /v1/public/premios?id_rede=uuid — catálogo para o app cliente (sem auth):
// prêmios ativos, dentro da vigência e com estoque (se houver controle de quantidade).
func (h *Handlers) PublicListarPremios(w http.ResponseWriter, r *http.Request) {
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
	itens, err := h.premioService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede valido")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("public listar premios: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar premios")
		}
		return
	}
	agora := time.Now()
	visiveis := filtrarPremiosPublicos(itens, agora)
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_rede": idRede,
		"itens":   visiveis,
		"total":   len(visiveis),
	})
}

func filtrarPremiosPublicos(itens []*modelos.Premio, agora time.Time) []*modelos.Premio {
	out := make([]*modelos.Premio, 0, len(itens))
	for _, p := range itens {
		if p == nil || !p.Ativo {
			continue
		}
		if agora.Before(p.VigenciaInicio) {
			continue
		}
		if p.VigenciaFim != nil && agora.After(*p.VigenciaFim) {
			continue
		}
		if p.QuantidadeDisponivel != nil && *p.QuantidadeDisponivel <= 0 {
			continue
		}
		out = append(out, p)
	}
	return out
}
