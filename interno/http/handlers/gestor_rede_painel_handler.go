package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// idRedeDaSessao retorna rede_id do JWT; o middleware da rota ja restringe o papel (gestor ou gerente de posto).
func (h *Handlers) idRedeDaSessao(w http.ResponseWriter, r *http.Request) (string, bool) {
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "sessao invalida")
		return "", false
	}
	id := strings.TrimSpace(u.IDRede)
	if id == "" {
		utils.ResponderErro(w, http.StatusBadRequest, "usuario sem rede vinculada")
		return "", false
	}
	return id, true
}

func (h *Handlers) MinhaRedeGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	rede, err := h.redeService.BuscarPorID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"rede": rede,
	})
}

func (h *Handlers) ListarGestoresDaMinhaRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	todos, err := h.gestorService.Listar()
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar gestores")
		return
	}
	var filtrados []*modelos.GestorRede
	for _, g := range todos {
		if strings.TrimSpace(g.IDRede) == idRede {
			filtrados = append(filtrados, g)
		}
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens": filtrados,
		"total": len(filtrados),
	})
}

func (h *Handlers) ListarCampanhasGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	itens, err := h.campanhaService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "rede invalida")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("listar campanhas gestor: %v", err)
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

func (h *Handlers) CriarCampanhaGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqCriarCampanha
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = idRede
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
		ModalidadeDesconto:  req.ModalidadeDesconto,
		BaseDesconto:        req.BaseDesconto,
		ValorDesconto:       req.ValorDesconto,
		ValorMinimoCompra:   req.ValorMinimoCompra,
		ValorMaximoCompra:   req.ValorMaximoCompra,
		MaxUsosPorCliente:   req.MaxUsosPorCliente,
		LitrosMin:           req.LitrosMin,
		LitrosMax:           req.LitrosMax,
		IDsCombustiveisRede: req.IDsCombustiveisRede,
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
			log.Printf("criar campanha gestor: %v", err)
			utils.ResponderJSON(w, http.StatusInternalServerError, map[string]string{
				"erro":    "falha ao criar campanha",
				"detalhe": err.Error(),
			})
		}
		return
	}
	h.notificarClientesPushNovaCampanha(c)
	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "campanha criada com sucesso",
		"campanha": c,
	})
}

func (h *Handlers) EditarCampanhaGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqEditarCampanha
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = idRede
	vi, vf, err := parseVigencias(req.VigenciaInicio, req.VigenciaFim)
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "vigencia_inicio e vigencia_fim devem ser ISO8601 (RFC3339)")
		return
	}
	ant, err := h.campanhaService.BuscarPorIDeRede(req.ID, idRede)
	if err != nil {
		if errors.Is(err, repositorios.ErrCampanhaNaoEncontrada) {
			utils.ResponderErro(w, http.StatusNotFound, "campanha nao encontrada")
			return
		}
		log.Printf("editar campanha gestor: buscar: %v", err)
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
		ModalidadeDesconto:  req.ModalidadeDesconto,
		BaseDesconto:        req.BaseDesconto,
		ValorDesconto:       req.ValorDesconto,
		ValorMinimoCompra:   req.ValorMinimoCompra,
		ValorMaximoCompra:   req.ValorMaximoCompra,
		MaxUsosPorCliente:   req.MaxUsosPorCliente,
		LitrosMin:           req.LitrosMin,
		LitrosMax:           req.LitrosMax,
		IDsCombustiveisRede: req.IDsCombustiveisRede,
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
			log.Printf("editar campanha gestor: %v", err)
			utils.ResponderJSON(w, http.StatusInternalServerError, map[string]string{
				"erro":    "falha ao atualizar campanha",
				"detalhe": err.Error(),
			})
		}
		return
	}
	nova, errC := h.campanhaService.BuscarPorIDeRede(req.ID, idRede)
	if errC == nil {
		h.notificarClientesSeCampanhaAtivada(ant, nova)
	} else {
		log.Printf("editar campanha gestor: recarregar apos gravar: %v", errC)
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "campanha atualizada com sucesso",
	})
}

func (h *Handlers) ListarPostosGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	itens, err := h.postoService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "rede invalida")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar postos")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens": itens,
		"total": len(itens),
	})
}

func (h *Handlers) CriarPostoGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqCriarPosto
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = idRede
	p, err := h.postoService.CriarPostoNaRede(&modelos.Posto{
		IDRede:       req.IDRede,
		Nome:         req.Nome,
		Codigo:       req.Codigo,
		NomeFantasia: req.NomeFantasia,
		CNPJ:         req.CNPJ,
		LogoURL:      req.LogoURL,
		Rua:          req.Rua,
		Numero:       req.Numero,
		Bairro:       req.Bairro,
		Complemento:  req.Complemento,
		CEP:          req.CEP,
		Cidade:       req.Cidade,
		Estado:       req.Estado,
		Telefone:     req.Telefone,
		EmailContato: req.EmailContato,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos: verifique nome, codigo, cnpj (14 digitos se informado), cep (8 digitos se informado), UF com 2 letras e URL do logo (http/https)")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case errors.Is(err, repositorios.ErrCodigoPostoDuplicadoNaRede):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		case errors.Is(err, repositorios.ErrCNPJPostoDuplicado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao criar posto")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "posto criado com sucesso",
		"posto":    p,
	})
}

func (h *Handlers) EditarPostoGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqEditarPosto
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = idRede
	p, err := h.postoService.AtualizarPostoNaRede(&modelos.Posto{
		ID:           req.ID,
		IDRede:       req.IDRede,
		Nome:         req.Nome,
		Codigo:       req.Codigo,
		NomeFantasia: req.NomeFantasia,
		CNPJ:         req.CNPJ,
		LogoURL:      req.LogoURL,
		Rua:          req.Rua,
		Numero:       req.Numero,
		Bairro:       req.Bairro,
		Complemento:  req.Complemento,
		CEP:          req.CEP,
		Cidade:       req.Cidade,
		Estado:       req.Estado,
		Telefone:     req.Telefone,
		EmailContato: req.EmailContato,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos: verifique nome, codigo, cnpj (14 digitos se informado), cep (8 digitos se informado), UF com 2 letras e URL do logo (http/https)")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case errors.Is(err, repositorios.ErrPostoNaoEncontradoNaRede):
			utils.ResponderErro(w, http.StatusNotFound, "posto nao encontrado nesta rede")
		case errors.Is(err, repositorios.ErrCodigoPostoDuplicadoNaRede):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		case errors.Is(err, repositorios.ErrCNPJPostoDuplicado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao atualizar posto")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "posto atualizado com sucesso",
		"posto":    p,
	})
}

func (h *Handlers) ListarPremiosGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	itens, err := h.premioService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "rede invalida")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("listar premios gestor: %v", err)
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

func (h *Handlers) CriarPremioGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqCriarPremio
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = idRede
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
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("criar premio gestor: %v", err)
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

func (h *Handlers) EditarPremioGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqEditarPremio
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = idRede
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
			log.Printf("editar premio gestor: %v", err)
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

func (h *Handlers) EditarMoedaVirtualMinhaRedeGestor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqEditarMoedaVirtualRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.ID = idRede
	rede, err := h.redeService.EditarMoedaVirtual(servicos.EditarMoedaVirtualRedeInput{
		ID:                  req.ID,
		MoedaVirtualNome:    req.MoedaVirtualNome,
		MoedaVirtualCotacao: req.MoedaVirtualCotacao,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe nome da moeda e cotacao maior que zero")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao atualizar moeda virtual")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "moeda virtual atualizada com sucesso",
		"rede":     rede,
	})
}

// EditarVoucherConfigMinhaRedeGestor PATCH /v1/gestor-rede/dev/redes/config-voucher
func (h *Handlers) EditarVoucherConfigMinhaRedeGestor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqEditarVoucherConfigRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	rede, err := h.redeService.EditarVoucherConfig(servicos.EditarVoucherConfigRedeInput{
		ID:      idRede,
		Dias:    req.Dias,
		Minutos: req.Minutos,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe ao menos um campo; dias 1-365; minutos PIX 5-10080")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao atualizar configuracao de voucher")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "configuracao de voucher atualizada",
		"rede":     rede,
	})
}

// EditarAppModulosMinhaRedeGestor PATCH /v1/gestor-rede/dev/redes/app-modulos
func (h *Handlers) EditarAppModulosMinhaRedeGestor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqEditarAppModulosRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	rede, err := h.redeService.EditarAppModulos(servicos.EditarAppModulosRedeInput{
		ID:                     idRede,
		AppModuloIndiqueGanhe:  req.AppModuloIndiqueGanhe,
		AppModuloCheckinDiario: req.AppModuloCheckinDiario,
		AppModuloGireGanhe:     req.AppModuloGireGanhe,
		AppModuloRedesSociais:  req.AppModuloRedesSociais,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao atualizar modulos do app")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "modulos do app atualizados",
		"rede":     rede,
	})
}

func (h *Handlers) ListarUsuariosRedeGestor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	limite := 0
	if v := strings.TrimSpace(q.Get("limite")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			utils.ResponderErro(w, http.StatusBadRequest, "parametro limite invalido")
			return
		}
		limite = n
	}
	offset := 0
	if v := strings.TrimSpace(q.Get("offset")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			utils.ResponderErro(w, http.StatusBadRequest, "parametro offset invalido")
			return
		}
		offset = n
	}
	var papeisFiltro []string
	if raw := strings.TrimSpace(q.Get("papeis")); raw != "" {
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				papeisFiltro = append(papeisFiltro, p)
			}
		}
	}
	idPosto := strings.TrimSpace(q.Get("id_posto"))
	itens, total, limiteEfetivo, offsetEfetivo, err := h.usuarioRedeService.ListarPorRedeIDPaginado(idRede, limite, offset, papeisFiltro, idPosto)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "parametros invalidos")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar usuarios da rede")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens":  itens,
		"total":  total,
		"limite": limiteEfetivo,
		"offset": offsetEfetivo,
	})
}

func (h *Handlers) CriarUsuarioEquipeGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqCriarUsuarioEquipe
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = idRede
	u, err := h.usuarioRedeService.CriarUsuarioEquipe(servicos.CriarUsuarioEquipeInput{
		IDRede:         req.IDRede,
		IDPosto:        req.IDPosto,
		Papel:          req.Papel,
		Nome:           req.Nome,
		Email:          req.Email,
		Senha:          req.Senha,
		ConfirmarSenha: req.ConfirmarSenha,
		Telefone:       req.Telefone,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case errors.Is(err, repositorios.ErrEmailUsuarioEquipeDuplicado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		case errors.Is(err, repositorios.ErrPostoNaoPertenceARede):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao criar usuario da equipe")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "usuario criado com sucesso",
		"usuario":  u,
	})
}

func (h *Handlers) EditarUsuarioEquipeGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqEditarUsuarioEquipe
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = idRede
	u, err := h.usuarioRedeService.EditarUsuarioEquipe(servicos.EditarUsuarioEquipeInput{
		IDRede:         req.IDRede,
		IDUsuario:      req.ID,
		IDPosto:        req.IDPosto,
		Papel:          req.Papel,
		Nome:           req.Nome,
		Email:          req.Email,
		Senha:          req.Senha,
		ConfirmarSenha: req.ConfirmarSenha,
		Telefone:       req.Telefone,
		Ativo:          req.Ativo,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		case errors.Is(err, repositorios.ErrUsuarioEquipeNaoEncontrado):
			utils.ResponderErro(w, http.StatusNotFound, "usuario da equipe nao encontrado nesta rede")
		case errors.Is(err, repositorios.ErrEmailUsuarioEquipeDuplicado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		case errors.Is(err, repositorios.ErrPostoNaoPertenceARede):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrDadosInvalidosUsuarioEquipe):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao atualizar usuario da equipe")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "usuario atualizado com sucesso",
		"usuario":  u,
	})
}

func (h *Handlers) ResumoRelatoriosGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	rede, err := h.redeService.BuscarPorID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar rede")
		}
		return
	}
	postos, err := h.postoService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "rede invalida")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("resumo relatorios gestor postos: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao agregar postos")
		}
		return
	}
	campanhas, err := h.campanhaService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "rede invalida")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("resumo relatorios gestor campanhas: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao agregar campanhas")
		}
		return
	}
	premios, err := h.premioService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "rede invalida")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("resumo relatorios gestor premios: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao agregar premios")
		}
		return
	}
	var campAtivas, campRascunho, campPausadas, campArquivadas int
	for _, c := range campanhas {
		switch c.Status {
		case modelos.StatusCampanhaAtiva:
			campAtivas++
		case modelos.StatusCampanhaRascunho:
			campRascunho++
		case modelos.StatusCampanhaPausada:
			campPausadas++
		case modelos.StatusCampanhaArquivada:
			campArquivadas++
		default:
			campArquivadas++
		}
	}
	premiosAtivos := 0
	for _, p := range premios {
		if p.Ativo {
			premiosAtivos++
		}
	}
	_, totalUsuarios, _, _, err := h.usuarioRedeService.ListarPorRedeIDPaginado(idRede, 1, 0, nil, "")
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "parametros invalidos")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("resumo relatorios gestor usuarios total: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao agregar usuarios")
		}
		return
	}
	_, totalClientes, _, _, err := h.usuarioRedeService.ListarPorRedeIDPaginado(idRede, 1, 0, []string{"cliente"}, "")
	if err != nil {
		log.Printf("resumo relatorios gestor clientes: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao agregar usuarios")
		return
	}
	_, totalEquipe, _, _, err := h.usuarioRedeService.ListarPorRedeIDPaginado(idRede, 1, 0, []string{"gerente_posto", "frentista"}, "")
	if err != nil {
		log.Printf("resumo relatorios gestor equipe: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao agregar usuarios")
		return
	}
	_, totalGestoresUsu, _, _, err := h.usuarioRedeService.ListarPorRedeIDPaginado(idRede, 1, 0, []string{"gestor_rede"}, "")
	if err != nil {
		log.Printf("resumo relatorios gestor gestores usu: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao agregar usuarios")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"resumo": map[string]any{
			"rede": map[string]any{
				"nome_fantasia":         rede.NomeFantasia,
				"moeda_virtual_nome":    rede.MoedaVirtualNome,
				"moeda_virtual_cotacao": rede.MoedaVirtualCotacao,
			},
			"postos": len(postos),
			"campanhas": map[string]any{
				"total":      len(campanhas),
				"ativas":     campAtivas,
				"rascunho":   campRascunho,
				"pausadas":   campPausadas,
				"arquivadas": campArquivadas,
			},
			"premios": map[string]any{
				"total":  len(premios),
				"ativos": premiosAtivos,
			},
			"usuarios": map[string]any{
				"total":         totalUsuarios,
				"clientes":      totalClientes,
				"equipe_postos": totalEquipe,
				"gestores_rede": totalGestoresUsu,
			},
		},
	})
}

func (h *Handlers) ListarAuditoriaGestorRede(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	limite := 50
	if v := strings.TrimSpace(q.Get("limite")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			utils.ResponderErro(w, http.StatusBadRequest, "parametro limite invalido (1 a 200)")
			return
		}
		limite = n
	}
	offset := 0
	if v := strings.TrimSpace(q.Get("offset")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			utils.ResponderErro(w, http.StatusBadRequest, "parametro offset invalido")
			return
		}
		offset = n
	}
	itens, total, err := h.auditoriaRepo.ListarPorRedeID(idRede, limite, offset)
	if err != nil {
		switch {
		case errors.Is(err, repositorios.ErrAuditoriaRedeIDInvalido):
			utils.ResponderErro(w, http.StatusBadRequest, "rede invalida")
		default:
			log.Printf("listar auditoria gestor: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar auditoria")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens":  itens,
		"total":  total,
		"limite": limite,
		"offset": offset,
	})
}

// IndiqueGanheConfigGestor GET/PATCH /v1/gestor-rede/dev/redes/indique-ganhe
func (h *Handlers) IndiqueGanheConfigGestor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getIndiqueGanheConfigGestor(w, r)
	case http.MethodPatch:
		h.patchIndiqueGanheConfigGestor(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) getIndiqueGanheConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.indiqueGanhe == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	cfg, err := h.indiqueGanhe.BuscarConfigIndique(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"regra":                   cfg.Regra,
		"moedas_premio_referente": cfg.MoedasPremioReferente,
		"moedas_premio_indicado":  cfg.MoedasPremioIndicado,
	})
}

type reqIndiqueGanheConfig struct {
	Regra                 string  `json:"regra"`
	MoedasPremioReferente float64 `json:"moedas_premio_referente"`
	MoedasPremioIndicado  float64 `json:"moedas_premio_indicado"`
}

func (h *Handlers) patchIndiqueGanheConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.indiqueGanhe == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqIndiqueGanheConfig
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	if err := h.indiqueGanhe.SalvarConfigIndique(idRede, req.Regra, req.MoedasPremioReferente, req.MoedasPremioIndicado); err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, "regra (CADASTRAR ou PRIMEIRA_COMPRA_VOUCHER) e moedas >= 0")
		} else {
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar")
		}
		return
	}
	cfg, _ := h.indiqueGanhe.BuscarConfigIndique(idRede)
	if cfg == nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]any{
			"mensagem": "configuracao do indique e ganhe salva",
		})
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem":                "configuracao do indique e ganhe salva",
		"regra":                   cfg.Regra,
		"moedas_premio_referente": cfg.MoedasPremioReferente,
		"moedas_premio_indicado":  cfg.MoedasPremioIndicado,
	})
}
