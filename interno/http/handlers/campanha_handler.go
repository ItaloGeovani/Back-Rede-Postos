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

type reqCriarCampanha struct {
	IDRede                string   `json:"id_rede"`
	Nome                  string   `json:"nome"`
	Titulo                string   `json:"titulo"`
	Descricao             string   `json:"descricao"`
	ImagemURL             string   `json:"imagem_url"`
	IDPosto               string   `json:"id_posto"`
	VigenciaInicio        *string  `json:"vigencia_inicio"`
	VigenciaFim           *string  `json:"vigencia_fim"`
	Status                string   `json:"status"`
	ValidaNoApp           *bool    `json:"valida_no_app"`
	ValidaNoPostoFisico   *bool    `json:"valida_no_posto_fisico"`
	TipoBeneficio         string   `json:"tipo_beneficio"`
	ModalidadeDesconto    string   `json:"modalidade_desconto"`
	BaseDesconto          string   `json:"base_desconto"`
	ValorDesconto         float64    `json:"valor_desconto"`
	ValorMinimoCompra     float64    `json:"valor_minimo_compra"`
	ValorMaximoCompra     *float64   `json:"valor_maximo_compra"`
	MaxUsosPorCliente     *int       `json:"max_usos_por_cliente"`
	LitrosMin             *float64   `json:"litros_min"`
	LitrosMax             *float64   `json:"litros_max"`
	IDsCombustiveisRede   []string   `json:"ids_combustiveis_rede"`
}

type reqEditarCampanha struct {
	ID                  string   `json:"id"`
	IDRede              string   `json:"id_rede"`
	Nome                string   `json:"nome"`
	Titulo              string   `json:"titulo"`
	Descricao           string   `json:"descricao"`
	ImagemURL           string   `json:"imagem_url"`
	IDPosto             string   `json:"id_posto"`
	VigenciaInicio      *string  `json:"vigencia_inicio"`
	VigenciaFim         *string  `json:"vigencia_fim"`
	Status              string   `json:"status"`
	ValidaNoApp         *bool    `json:"valida_no_app"`
	ValidaNoPostoFisico *bool    `json:"valida_no_posto_fisico"`
	TipoBeneficio       string   `json:"tipo_beneficio"`
	ModalidadeDesconto  string   `json:"modalidade_desconto"`
	BaseDesconto        string   `json:"base_desconto"`
	ValorDesconto         float64    `json:"valor_desconto"`
	ValorMinimoCompra     float64    `json:"valor_minimo_compra"`
	ValorMaximoCompra     *float64   `json:"valor_maximo_compra"`
	MaxUsosPorCliente     *int       `json:"max_usos_por_cliente"`
	LitrosMin             *float64   `json:"litros_min"`
	LitrosMax             *float64   `json:"litros_max"`
	IDsCombustiveisRede   []string   `json:"ids_combustiveis_rede"`
}

func boolOuPadrao(v *bool, padrao bool) bool {
	if v == nil {
		return padrao
	}
	return *v
}

func (h *Handlers) ListarCampanhasRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	itens, err := h.campanhaService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede valido")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("listar campanhas: %v", err)
			utils.ResponderJSON(w, http.StatusInternalServerError, map[string]string{
				"erro":    "falha ao listar campanhas",
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

func (h *Handlers) CriarCampanhaRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqCriarCampanha
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}

	u := middlewares.Usuario(r.Context())
	if u == nil || strings.TrimSpace(u.IDUsuario) == "" {
		utils.ResponderErro(w, http.StatusUnauthorized, "sessao invalida")
		return
	}

	vi, vf, err := parseVigencias(req.VigenciaInicio, req.VigenciaFim)
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "vigencia_inicio e vigencia_fim devem ser ISO8601 (RFC3339)")
		return
	}

	c, err := h.campanhaService.Criar(u.IDUsuario, servicos.CriarCampanhaInput{
		IDRede:                req.IDRede,
		Nome:                  req.Nome,
		Titulo:                req.Titulo,
		Descricao:             req.Descricao,
		ImagemURL:             req.ImagemURL,
		IDPosto:               req.IDPosto,
		VigenciaInicio:        vi,
		VigenciaFim:           vf,
		Status:                modelos.StatusCampanha(strings.TrimSpace(req.Status)),
		ValidaNoApp:           boolOuPadrao(req.ValidaNoApp, true),
		ValidaNoPostoFisico:   boolOuPadrao(req.ValidaNoPostoFisico, false),
		TipoBeneficio:         req.TipoBeneficio,
		ModalidadeDesconto:    req.ModalidadeDesconto,
		BaseDesconto:          req.BaseDesconto,
		ValorDesconto:         req.ValorDesconto,
		ValorMinimoCompra:     req.ValorMinimoCompra,
		ValorMaximoCompra:     req.ValorMaximoCompra,
		MaxUsosPorCliente:     req.MaxUsosPorCliente,
		LitrosMin:             req.LitrosMin,
		LitrosMax:             req.LitrosMax,
		IDsCombustiveisRede:   req.IDsCombustiveisRede,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos: canal exclusivo (app OU posto fisico), vigencias, desconto, valor minimo ou limite de usos")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case errors.Is(err, repositorios.ErrPostoNaoPertenceARede):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		default:
			log.Printf("criar campanha: %v", err)
			utils.ResponderJSON(w, http.StatusInternalServerError, map[string]string{
				"erro":    "falha ao criar campanha",
				"detalhe": err.Error(),
			})
		}
		return
	}
	h.registrarEventoCampanhaCriada(c)
	push := h.notificarClientesPushNovaCampanha(c)
	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "campanha criada com sucesso",
		"campanha": c,
		"push":     push,
	})
}

func (h *Handlers) EditarCampanhaRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqEditarCampanha
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}

	vi, vf, err := parseVigencias(req.VigenciaInicio, req.VigenciaFim)
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "vigencia_inicio e vigencia_fim devem ser ISO8601 (RFC3339)")
		return
	}
	idRede := strings.TrimSpace(req.IDRede)
	if idRede == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "id_rede obrigatorio")
		return
	}
	ant, err := h.campanhaService.BuscarPorIDeRede(req.ID, idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrCampanhaNaoEncontrada) {
			utils.ResponderErro(w, http.StatusNotFound, "campanha nao encontrada")
			return
		}
		log.Printf("editar campanha dev: buscar: %v", err)
		utils.ResponderJSON(w, http.StatusInternalServerError, map[string]string{
			"erro":    "falha ao carregar campanha",
			"detalhe": err.Error(),
		})
		return
	}

	err = h.campanhaService.Atualizar(servicos.AtualizarCampanhaInput{
		ID:                  req.ID,
		IDRede:              req.IDRede,
		Nome:                req.Nome,
		Titulo:              req.Titulo,
		Descricao:           req.Descricao,
		ImagemURL:           req.ImagemURL,
		IDPosto:             req.IDPosto,
		VigenciaInicio:      vi,
		VigenciaFim:         vf,
		Status:              modelos.StatusCampanha(strings.TrimSpace(req.Status)),
		ValidaNoApp:         boolOuPadrao(req.ValidaNoApp, true),
		ValidaNoPostoFisico: boolOuPadrao(req.ValidaNoPostoFisico, false),
		TipoBeneficio:       req.TipoBeneficio,
		ModalidadeDesconto:  req.ModalidadeDesconto,
		BaseDesconto:        req.BaseDesconto,
		ValorDesconto:         req.ValorDesconto,
		ValorMinimoCompra:     req.ValorMinimoCompra,
		ValorMaximoCompra:     req.ValorMaximoCompra,
		MaxUsosPorCliente:     req.MaxUsosPorCliente,
		LitrosMin:             req.LitrosMin,
		LitrosMax:             req.LitrosMax,
		IDsCombustiveisRede:   req.IDsCombustiveisRede,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case errors.Is(err, repositorios.ErrCampanhaNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "campanha nao encontrada")
		case errors.Is(err, repositorios.ErrPostoNaoPertenceARede):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		default:
			log.Printf("editar campanha: %v", err)
			utils.ResponderJSON(w, http.StatusInternalServerError, map[string]string{
				"erro":    "falha ao atualizar campanha",
				"detalhe": err.Error(),
			})
		}
		return
	}

	nova, errC := h.campanhaService.BuscarPorIDeRede(req.ID, idRede)
	var push ResultadoPushCampanha
	if errC == nil {
		push = h.notificarClientesSeCampanhaAtivada(ant, nova)
		h.registrarEventoCampanhaAtivada(ant, nova)
	} else {
		log.Printf("editar campanha dev: recarregar apos gravar: %v", errC)
		push = ResultadoPushCampanha{Disparado: false, Motivo: "falha ao recarregar campanha apos gravar"}
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "campanha atualizada com sucesso",
		"push":     push,
	})
}

func parseVigencias(ini, fim *string) (*time.Time, *time.Time, error) {
	if ini == nil || strings.TrimSpace(*ini) == "" {
		return nil, nil, errors.New("vigencia_inicio obrigatoria")
	}
	if fim == nil || strings.TrimSpace(*fim) == "" {
		return nil, nil, errors.New("vigencia_fim obrigatoria")
	}
	tIni, err := time.Parse(time.RFC3339, strings.TrimSpace(*ini))
	if err != nil {
		return nil, nil, err
	}
	tFim, err := time.Parse(time.RFC3339, strings.TrimSpace(*fim))
	if err != nil {
		return nil, nil, err
	}
	tIniPtr := &tIni
	tFimPtr := &tFim
	return tIniPtr, tFimPtr, nil
}

// PublicListarCampanhas GET /v1/public/campanhas?id_rede=uuid — campanhas ativas para o app cliente (sem auth).
func (h *Handlers) PublicListarCampanhas(w http.ResponseWriter, r *http.Request) {
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
	itens, err := h.campanhaService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede valido")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("public listar campanhas: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar campanhas")
		}
		return
	}
	agora := time.Now()
	visiveis := filtrarCampanhasPublicas(itens, agora)
	publicos := make([]*modelos.Campanha, 0, len(visiveis))
	for _, c := range visiveis {
		cp := *c
		cp.CriadoPor = ""
		publicos = append(publicos, &cp)
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_rede": idRede,
		"itens":   publicos,
		"total":   len(publicos),
	})
}

func filtrarCampanhasPublicas(itens []*modelos.Campanha, agora time.Time) []*modelos.Campanha {
	out := make([]*modelos.Campanha, 0, len(itens))
	for _, c := range itens {
		if c == nil || c.Status != modelos.StatusCampanhaAtiva {
			continue
		}
		if !c.ValidaNoApp {
			continue
		}
		if c.VigenciaInicio != nil && agora.Before(*c.VigenciaInicio) {
			continue
		}
		if c.VigenciaFim != nil && agora.After(*c.VigenciaFim) {
			continue
		}
		out = append(out, c)
	}
	return out
}
