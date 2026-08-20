package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// idPostoCombustivelDaSessao: gerente usa o posto da sessão; gestor usa query/body.
func (h *Handlers) idPostoCombustivelDaSessao(w http.ResponseWriter, r *http.Request, idPostoReq string) (string, bool) {
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "sessao invalida")
		return "", false
	}
	if u.Papel == modelos.PapelGerentePosto {
		id := strings.TrimSpace(u.IDPosto)
		if id == "" {
			utils.ResponderErro(w, http.StatusBadRequest, "gerente sem posto vinculado")
			return "", false
		}
		return id, true
	}
	id := strings.TrimSpace(idPostoReq)
	if id == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_posto")
		return "", false
	}
	return id, true
}

// ListarCombustiveisRede GET .../combustiveis/listar?id_posto=
func (h *Handlers) ListarCombustiveisRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	if h.combustivelRedeService == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	idPosto, ok := h.idPostoCombustivelDaSessao(w, r, r.URL.Query().Get("id_posto"))
	if !ok {
		return
	}
	itens, err := h.combustivelRedeService.Listar(idRede, idPosto)
	if err != nil {
		switch {
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "rede ou posto invalido")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar combustiveis")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens":    itens,
		"total":    len(itens),
		"id_posto": idPosto,
	})
}

type reqCriarCombustivelRede struct {
	IDPosto       string  `json:"id_posto"`
	Nome          string  `json:"nome"`
	Codigo        string  `json:"codigo"`
	Descricao     string  `json:"descricao"`
	PrecoPorLitro float64 `json:"preco_por_litro"`
	Ordem         int     `json:"ordem"`
	Ativo         *bool   `json:"ativo"`
}

// CriarCombustivelRede POST .../combustiveis/criar
func (h *Handlers) CriarCombustivelRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	if h.combustivelRedeService == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	var req reqCriarCombustivelRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	idPosto, ok := h.idPostoCombustivelDaSessao(w, r, req.IDPosto)
	if !ok {
		return
	}
	ativo := true
	if req.Ativo != nil {
		ativo = *req.Ativo
	}
	reg, err := h.combustivelRedeService.Criar(idRede, servicos.CriarCombustivelRedeInput{
		IDPosto:       idPosto,
		Nome:          req.Nome,
		Codigo:        req.Codigo,
		Descricao:     req.Descricao,
		PrecoPorLitro: req.PrecoPorLitro,
		Ordem:         req.Ordem,
		Ativo:         ativo,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_posto, nome e preco por litro (>= 0)")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case errors.Is(err, repositorios.ErrPostoNaoPertenceARede):
			utils.ResponderErro(w, http.StatusBadRequest, "posto nao pertence a esta rede")
		case strings.Contains(err.Error(), "ja existe"):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao criar combustivel")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem":    "combustivel criado",
		"combustivel": reg,
	})
}

type reqEditarCombustivelRede struct {
	ID            string  `json:"id"`
	Nome          string  `json:"nome"`
	Codigo        string  `json:"codigo"`
	Descricao     string  `json:"descricao"`
	PrecoPorLitro float64 `json:"preco_por_litro"`
	Ordem         int     `json:"ordem"`
	Ativo         *bool   `json:"ativo"`
}

// EditarCombustivelRede PATCH .../combustiveis/editar
func (h *Handlers) EditarCombustivelRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	if h.combustivelRedeService == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	var req reqEditarCombustivelRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	u := middlewares.Usuario(r.Context())
	if u != nil && u.Papel == modelos.PapelGerentePosto {
		idPostoSessao := strings.TrimSpace(u.IDPosto)
		if idPostoSessao == "" {
			utils.ResponderErro(w, http.StatusBadRequest, "gerente sem posto vinculado")
			return
		}
		atual, err := h.combustivelRedeService.BuscarPorID(req.ID, idRede, idPostoSessao)
		if err != nil {
			if errors.Is(err, repositorios.ErrCombustivelRedeNaoEncontrado) {
				utils.ResponderErro(w, http.StatusNotFound, "combustivel nao encontrado")
				return
			}
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao atualizar combustivel")
			return
		}
		_ = atual
	}
	ativo := true
	if req.Ativo != nil {
		ativo = *req.Ativo
	}
	reg, err := h.combustivelRedeService.Atualizar(idRede, servicos.AtualizarCombustivelRedeInput{
		ID:            req.ID,
		Nome:          req.Nome,
		Codigo:        req.Codigo,
		Descricao:     req.Descricao,
		PrecoPorLitro: req.PrecoPorLitro,
		Ordem:         req.Ordem,
		Ativo:         ativo,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe id, nome e preco por litro (>= 0)")
		case errors.Is(err, repositorios.ErrCombustivelRedeNaoEncontrado):
			utils.ResponderErro(w, http.StatusNotFound, "combustivel nao encontrado")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case strings.Contains(err.Error(), "ja existe"):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao atualizar combustivel")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":    "combustivel atualizado",
		"combustivel": reg,
	})
}

// ExcluirCombustivelRede DELETE .../combustiveis/excluir?id=uuid
func (h *Handlers) ExcluirCombustivelRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	if h.combustivelRedeService == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u != nil && u.Papel == modelos.PapelGerentePosto {
		idPostoSessao := strings.TrimSpace(u.IDPosto)
		if idPostoSessao == "" {
			utils.ResponderErro(w, http.StatusBadRequest, "gerente sem posto vinculado")
			return
		}
		if _, err := h.combustivelRedeService.BuscarPorID(id, idRede, idPostoSessao); err != nil {
			if errors.Is(err, repositorios.ErrCombustivelRedeNaoEncontrado) {
				utils.ResponderErro(w, http.StatusNotFound, "combustivel nao encontrado")
				return
			}
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao excluir combustivel")
			return
		}
	}
	err := h.combustivelRedeService.Excluir(id, idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "id invalido")
		case errors.Is(err, repositorios.ErrCombustivelRedeNaoEncontrado):
			utils.ResponderErro(w, http.StatusNotFound, "combustivel nao encontrado")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao excluir combustivel")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "combustivel excluido",
	})
}

// PublicListarCombustiveisRede GET /v1/public/rede-combustiveis?id_rede=&id_posto= — ativos do posto.
func (h *Handlers) PublicListarCombustiveisRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.combustivelRedeService == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	idPosto := strings.TrimSpace(r.URL.Query().Get("id_posto"))
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	if idPosto == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_posto")
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
	itens, err := h.combustivelRedeService.Listar(idRede, idPosto)
	if err != nil {
		log.Printf("public listar combustiveis: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar combustiveis")
		return
	}
	ativos := make([]*repositorios.CombustivelRedeRegistro, 0, len(itens))
	for _, c := range itens {
		if c != nil && c.Ativo {
			ativos = append(ativos, c)
		}
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_rede":  idRede,
		"id_posto": idPosto,
		"itens":    ativos,
		"total":    len(ativos),
	})
}
