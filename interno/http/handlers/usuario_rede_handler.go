package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type reqLoginUsuarioPainel struct {
	IDRede string `json:"id_rede"`
	Email  string `json:"email"`
	Senha  string `json:"senha"`
}

type reqCriarUsuarioEquipe struct {
	IDRede         string `json:"id_rede"`
	IDPosto        string `json:"id_posto"`
	Papel          string `json:"papel"`
	Nome           string `json:"nome"`
	Email          string `json:"email"`
	Senha          string `json:"senha"`
	ConfirmarSenha string `json:"confirmar_senha"`
	Telefone       string `json:"telefone"`
}

type reqEditarUsuarioEquipe struct {
	ID             string `json:"id"`
	IDRede         string `json:"id_rede"`
	IDPosto        string `json:"id_posto"`
	Papel          string `json:"papel"`
	Nome           string `json:"nome"`
	Email          string `json:"email"`
	Senha          string `json:"senha"`
	ConfirmarSenha string `json:"confirmar_senha"`
	Telefone       string `json:"telefone"`
	Ativo          bool   `json:"ativo"`
}

func (h *Handlers) ListarUsuariosRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	q := r.URL.Query()
	idRede := strings.TrimSpace(q.Get("id_rede"))

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
			utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede valido")
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

func (h *Handlers) CriarUsuarioEquipeRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqCriarUsuarioEquipe
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}

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
		"mensagem": "usuario da equipe criado com sucesso",
		"usuario":  u,
	})
}

func (h *Handlers) EditarUsuarioEquipeRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqEditarUsuarioEquipe
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}

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
		"mensagem": "usuario da equipe atualizado com sucesso",
		"usuario":  u,
	})
}

func (h *Handlers) LoginUsuarioRedePainelDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqLoginUsuarioPainel
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "payload invalido")
		return
	}
	req.IDRede = strings.TrimSpace(req.IDRede)

	var token string
	var sessao *modelos.UsuarioSessao
	var err error
	if req.IDRede == "" {
		token, sessao, err = h.usuarioRedeService.LoginPainel(req.Email, req.Senha)
	} else {
		token, sessao, err = h.usuarioRedeService.LoginPainelNaRede(req.Email, req.Senha, req.IDRede)
	}
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, servicos.ErrCredenciais):
			utils.ResponderErro(w, http.StatusUnauthorized, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao autenticar usuario")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "login realizado com sucesso",
		"token":    token,
		"sessao":   sessao,
	})
}
