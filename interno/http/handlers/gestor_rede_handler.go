package handlers

import (
	"errors"
	"net/http"

	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

type reqCriarGestorComPlano struct {
	IDRede         string `json:"id_rede"`
	Nome           string `json:"nome"`
	Email          string `json:"email"`
	Senha          string `json:"senha"`
	ConfirmarSenha string `json:"confirmar_senha"`
	Telefone       string `json:"telefone"`
}

type reqEditarGestorComPlano struct {
	ID             string `json:"id"`
	Nome           string `json:"nome"`
	Email          string `json:"email"`
	Telefone       string `json:"telefone"`
	Ativo          bool   `json:"ativo"`
	Senha          string `json:"senha"`
	ConfirmarSenha string `json:"confirmar_senha"`
}

type reqLoginGestorRede struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

func (h *Handlers) ListarGestoresRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	itens, err := h.gestorService.Listar()
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar gestores da rede")
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"itens": itens,
		"total": len(itens),
	})
}

func (h *Handlers) CriarGestorRedeComPlanoDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqCriarGestorComPlano
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}

	gestor, proximosVencimentos, err := h.gestorService.CriarComPlano(servicos.CriarGestorComPlanoInput{
		IDRede:         req.IDRede,
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
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		case errors.Is(err, repositorios.ErrEmailGestorDuplicado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	utils.ResponderJSON(w, http.StatusCreated, map[string]any{
		"mensagem": "gestor da rede criado com sucesso",
		"gestor":   gestor,
		"cobranca": map[string]any{
			"valor_implantacao":    gestor.ValorImplantacao,
			"valor_mensalidade":    gestor.ValorMensalidade,
			"primeiro_vencimento":  gestor.PrimeiroVencimento.Format("2006-01-02"),
			"dia_recorrencia":      gestor.DiaVencimento,
			"proximos_vencimentos": proximosVencimentos,
		},
	})
}

func (h *Handlers) EditarGestorRedeComPlanoDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqEditarGestorComPlano
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}

	gestor, proximosVencimentos, err := h.gestorService.EditarComPlano(servicos.EditarGestorComPlanoInput{
		ID:             req.ID,
		Nome:           req.Nome,
		Email:          req.Email,
		Telefone:       req.Telefone,
		Ativo:          req.Ativo,
		Senha:          req.Senha,
		ConfirmarSenha: req.ConfirmarSenha,
	})
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositorios.ErrEmailGestorDuplicado):
			utils.ResponderErro(w, http.StatusConflict, err.Error())
		case errors.Is(err, repositorios.ErrGestorNaoEncontrado):
			utils.ResponderErro(w, http.StatusNotFound, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	cobranca := map[string]any{
		"valor_implantacao":    gestor.ValorImplantacao,
		"valor_mensalidade":    gestor.ValorMensalidade,
		"dia_recorrencia":      gestor.DiaVencimento,
		"proximos_vencimentos": proximosVencimentos,
	}
	if !gestor.PrimeiroVencimento.IsZero() {
		cobranca["primeiro_vencimento"] = gestor.PrimeiroVencimento.Format("2006-01-02")
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "gestor da rede atualizado",
		"gestor":   gestor,
		"cobranca": cobranca,
	})
}

func (h *Handlers) LoginGestorRedeDev(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}

	var req reqLoginGestorRede
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}

	token, sessao, err := h.gestorService.Login(req.Email, req.Senha)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, servicos.ErrCredenciais):
			utils.ResponderErro(w, http.StatusUnauthorized, err.Error())
		default:
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao autenticar gestor da rede")
		}
		return
	}

	utils.ResponderJSON(w, http.StatusOK, map[string]any{
		"mensagem": "login de gestor da rede realizado com sucesso",
		"token":    token,
		"sessao":   sessao,
	})
}
