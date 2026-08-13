package handlers

import (
	"errors"
	"net/http"

	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"
)

// NiveisClienteConfigGestor GET/PATCH /v1/gestor-rede/dev/redes/niveis-cliente
func (h *Handlers) NiveisClienteConfigGestor(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getNiveisClienteConfigGestor(w, r)
	case http.MethodPatch:
		h.patchNiveisClienteConfigGestor(w, r)
	default:
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
	}
}

func (h *Handlers) getNiveisClienteConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.niveisCliente == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	cfg, err := h.niveisCliente.Buscar(idRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao carregar configuracao")
		return
	}
	utils.ResponderJSON(w, http.StatusOK, cfg)
}

type reqNiveisCliente struct {
	Ativo             bool                         `json:"ativo"`
	MultDescontoAtivo bool                         `json:"mult_desconto_ativo"`
	Niveis            []repositorios.NivelClienteLinha `json:"niveis"`
}

func (h *Handlers) patchNiveisClienteConfigGestor(w http.ResponseWriter, r *http.Request) {
	if h.niveisCliente == nil {
		utils.ResponderErro(w, http.StatusNotImplemented, "indisponivel")
		return
	}
	idRede, ok := h.idRedeDaSessao(w, r)
	if !ok {
		return
	}
	var req reqNiveisCliente
	if err := utils.DecodificarJSON(r, &req); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, utils.MensagemDecodeJSON(err))
		return
	}
	if len(req.Niveis) == 0 {
		utils.ResponderErro(w, http.StatusBadRequest, "informe o array niveis")
		return
	}
	if err := h.niveisCliente.Salvar(idRede, req.Ativo, req.MultDescontoAtivo, req.Niveis); err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, err.Error())
		} else {
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao salvar")
		}
		return
	}
	cfg, _ := h.niveisCliente.Buscar(idRede)
	if cfg == nil {
		utils.ResponderJSON(w, http.StatusOK, map[string]string{"mensagem": "configuracao de niveis salva"})
		return
	}
	utils.ResponderJSON(w, http.StatusOK, cfg)
}
