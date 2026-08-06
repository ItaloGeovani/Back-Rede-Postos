package handlers

import (
	"encoding/json"
	"errors"
	"io"
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

type bodyPremioResgatar struct {
	PremioID string `json:"premio_id"`
}

type bodyPremioResgateID struct {
	ID     string `json:"id"`
	IDRede string `json:"id_rede,omitempty"`
	Motivo string `json:"motivo,omitempty"`
}

// PostEuPremioResgatar POST /v1/eu/premios/resgatar
func (h *Handlers) PostEuPremioResgatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.premioResgateSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes")
		return
	}
	var body bodyPremioResgatar
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "corpo json invalido")
		return
	}
	out, err := h.premioResgateSvc.Resgatar(u.IDRede, u.IDUsuario, body.PremioID)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos")
		case errors.Is(err, repositorios.ErrPremioNaoEncontrado):
			utils.ResponderErro(w, http.StatusNotFound, "premio nao encontrado")
		case errors.Is(err, servicos.ErrPremioIndisponivel), errors.Is(err, repositorios.ErrPremioEsgotado):
			utils.ResponderErro(w, http.StatusConflict, "premio indisponivel")
		case errors.Is(err, repositorios.ErrSaldoInsuficiente):
			utils.ResponderErro(w, http.StatusConflict, "saldo insuficiente")
		default:
			log.Printf("premio resgatar: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao resgatar premio")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

// GetEuPremioResgates GET /v1/eu/premios/resgates
func (h *Handlers) GetEuPremioResgates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.premioResgateSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil || u.Papel != modelos.PapelCliente {
		utils.ResponderErro(w, http.StatusForbidden, "apenas clientes")
		return
	}
	itens, err := h.premioResgateSvc.ListarMeus(u.IDRede, u.IDUsuario)
	if err != nil {
		log.Printf("premio listar meus: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar resgates")
		return
	}
	if itens == nil {
		itens = []*modelos.PremioResgate{}
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"itens": itens, "total": len(itens)})
}

func (h *Handlers) idRedePainelPremio(r *http.Request, bodyIDRede string) (string, error) {
	u := middlewares.Usuario(r.Context())
	if u == nil {
		return "", errors.New("nao autenticado")
	}
	if u.Papel == modelos.PapelSuperAdmin {
		id := strings.TrimSpace(bodyIDRede)
		if id == "" {
			id = strings.TrimSpace(r.URL.Query().Get("id_rede"))
		}
		if id == "" {
			return "", servicos.ErrDadosInvalidos
		}
		return id, nil
	}
	return strings.TrimSpace(u.IDRede), nil
}

// GetPremioResgatesPainel GET …/premios/resgates/listar
func (h *Handlers) GetPremioResgatesPainel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.premioResgateSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	idRede, err := h.idRedePainelPremio(r, "")
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limite, _ := strconv.Atoi(r.URL.Query().Get("limite"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	itens, total, err := h.premioResgateSvc.ListarRede(idRede, status, limite, offset)
	if err != nil {
		if errors.Is(err, servicos.ErrDadosInvalidos) {
			utils.ResponderErro(w, http.StatusBadRequest, "filtro invalido")
			return
		}
		log.Printf("premio resgates painel: %v", err)
		utils.ResponderErro(w, http.StatusInternalServerError, "falha ao listar resgates")
		return
	}
	if itens == nil {
		itens = []*modelos.PremioResgate{}
	}
	utils.ResponderJSON(w, http.StatusOK, map[string]any{"itens": itens, "total": total})
}

// PostPremioResgateEntregar POST …/premios/resgates/entregar
func (h *Handlers) PostPremioResgateEntregar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.premioResgateSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "nao autenticado")
		return
	}
	var body bodyPremioResgateID
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "corpo json invalido")
		return
	}
	idRede, err := h.idRedePainelPremio(r, body.IDRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	var posto *string
	if strings.TrimSpace(u.IDPosto) != "" {
		s := strings.TrimSpace(u.IDPosto)
		posto = &s
	}
	out, err := h.premioResgateSvc.Entregar(idRede, body.ID, posto, u)
	if err != nil {
		switch {
		case errors.Is(err, repositorios.ErrPremioResgateNaoEncontrado):
			utils.ResponderErro(w, http.StatusNotFound, "resgate nao encontrado")
		case errors.Is(err, repositorios.ErrPremioResgateStatus):
			utils.ResponderErro(w, http.StatusConflict, "resgate nao esta aguardando retirada")
		case errors.Is(err, servicos.ErrDadosInvalidos):
			utils.ResponderErro(w, http.StatusBadRequest, "dados invalidos")
		default:
			log.Printf("premio entregar: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao registrar entrega")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

// PostPremioResgateCancelar POST …/premios/resgates/cancelar
func (h *Handlers) PostPremioResgateCancelar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ResponderErro(w, http.StatusMethodNotAllowed, "metodo nao permitido")
		return
	}
	if h.premioResgateSvc == nil {
		utils.ResponderErro(w, http.StatusServiceUnavailable, "servico indisponivel")
		return
	}
	u := middlewares.Usuario(r.Context())
	if u == nil {
		utils.ResponderErro(w, http.StatusUnauthorized, "nao autenticado")
		return
	}
	var body bodyPremioResgateID
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&body); err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "corpo json invalido")
		return
	}
	idRede, err := h.idRedePainelPremio(r, body.IDRede)
	if err != nil {
		utils.ResponderErro(w, http.StatusBadRequest, "informe id_rede")
		return
	}
	out, err := h.premioResgateSvc.Cancelar(idRede, body.ID, body.Motivo, u)
	if err != nil {
		switch {
		case errors.Is(err, servicos.ErrPremioResgateCancelado):
			utils.ResponderErro(w, http.StatusForbidden, "sem permissao para cancelar")
		case errors.Is(err, repositorios.ErrPremioResgateNaoEncontrado):
			utils.ResponderErro(w, http.StatusNotFound, "resgate nao encontrado")
		case errors.Is(err, repositorios.ErrPremioResgateStatus):
			utils.ResponderErro(w, http.StatusConflict, "resgate nao esta aguardando retirada")
		default:
			log.Printf("premio cancelar: %v", err)
			utils.ResponderErro(w, http.StatusInternalServerError, "falha ao cancelar resgate")
		}
		return
	}
	utils.ResponderJSON(w, http.StatusOK, out)
}

// ListarPremiosFrentista GET — catálogo leitura (opcional para frentista).
func (h *Handlers) ListarPremiosFrentista(w http.ResponseWriter, r *http.Request) {
	h.ListarPremiosGestorRede(w, r)
}
