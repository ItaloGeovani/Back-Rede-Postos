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

func (h *Handlers) ListarPostosRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	idRede := strings.TrimSpace(r.URL.Query().Get("id_rede"))
	itens, err := h.postoService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede valido")
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

type reqCriarPosto struct {
	IDRede       string `json:"id_rede"`
	Nome         string `json:"nome"`
	Codigo       string `json:"codigo"`
	NomeFantasia string `json:"nome_fantasia"`
	CNPJ         string `json:"cnpj"`
	LogoURL      string `json:"logo_url"`
	Rua          string `json:"rua"`
	Numero       string `json:"numero"`
	Bairro       string `json:"bairro"`
	Complemento  string `json:"complemento"`
	CEP          string `json:"cep"`
	Cidade       string `json:"cidade"`
	Estado       string `json:"estado"`
	Telefone     string `json:"telefone"`
	EmailContato string `json:"email_contato"`
}

type reqEditarPosto struct {
	ID           string `json:"id"`
	IDRede       string `json:"id_rede"`
	Nome         string `json:"nome"`
	Codigo       string `json:"codigo"`
	NomeFantasia string `json:"nome_fantasia"`
	CNPJ         string `json:"cnpj"`
	LogoURL      string `json:"logo_url"`
	Rua          string `json:"rua"`
	Numero       string `json:"numero"`
	Bairro       string `json:"bairro"`
	Complemento  string `json:"complemento"`
	CEP          string `json:"cep"`
	Cidade       string `json:"cidade"`
	Estado       string `json:"estado"`
	Telefone     string `json:"telefone"`
	EmailContato string `json:"email_contato"`
}

func (h *Handlers) CriarPostoRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqCriarPosto
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}

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

func (h *Handlers) EditarPostoRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqEditarPosto
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}

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
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos: verifique id, id_rede, nome, codigo, cnpj (14 digitos se informado), cep (8 digitos se informado), UF com 2 letras e URL do logo (http/https)")
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

// PublicListarPostos GET /v1/public/postos?id_rede=uuid — postos da rede para o app cliente (sem auth).
func (h *Handlers) PublicListarPostos(w http.ResponseWriter, r *http.Request) {
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
	itens, err := h.postoService.ListarPorRedeID(idRede)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede valido")
		case errors.Is(err, repositorios.ErrRedeNaoEncontrada):
			utils.ResponderErro(w, http.StatusNotFound, "rede nao encontrada")
		default:
			log.Printf("public listar postos: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar postos")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"id_rede": idRede,
		"itens":   itens,
		"total":   len(itens),
	})
}
