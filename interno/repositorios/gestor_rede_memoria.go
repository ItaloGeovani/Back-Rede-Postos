package repositorios

import (
	"errors"
	"strings"
	"sync"
	"time"

	"gaspass-servidor/interno/modelos"
)

var (
	ErrGestorNaoEncontrado  = errors.New("gestor da rede nao encontrado")
	ErrEmailGestorDuplicado = errors.New("email do gestor ja cadastrado")
)

type GestorRedeRepositorio interface {
	Listar() ([]*modelos.GestorRede, error)
	Criar(gestor *modelos.GestorRede) error
	Atualizar(id string, atualizar func(*modelos.GestorRede) error) (*modelos.GestorRede, error)
	Contar() (total int, ativos int, err error)
	// BuscarPorEmailParaLogin retorna o registro com SenhaHash preenchido (uso interno / autenticacao).
	BuscarPorEmailParaLogin(email string) (*modelos.GestorRede, error)
	ExisteGestorPorEmail(email string) (bool, error)
}

type gestorRedeMemoria struct {
	mu      sync.RWMutex
	porID   map[string]*modelos.GestorRede
	porMail map[string]string
}

func NovoGestorRedeMemoria() GestorRedeRepositorio {
	return &gestorRedeMemoria{
		porID:   make(map[string]*modelos.GestorRede),
		porMail: make(map[string]string),
	}
}

func (r *gestorRedeMemoria) Listar() ([]*modelos.GestorRede, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lista := make([]*modelos.GestorRede, 0, len(r.porID))
	for _, item := range r.porID {
		lista = append(lista, clonarGestor(item))
	}
	return lista, nil
}

func (r *gestorRedeMemoria) Criar(gestor *modelos.GestorRede) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	emailNorm := normalizarEmailGestor(gestor.Email)
	if _, ok := r.porMail[emailNorm]; ok {
		return ErrEmailGestorDuplicado
	}

	copia := *gestor
	copia.Email = emailNorm
	now := time.Now().UTC()
	copia.CriadoEm = now
	copia.AtualizadoEm = now

	r.porID[copia.ID] = &copia
	r.porMail[emailNorm] = copia.ID
	return nil
}

func (r *gestorRedeMemoria) Atualizar(id string, atualizar func(*modelos.GestorRede) error) (*modelos.GestorRede, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	gestor, ok := r.porID[id]
	if !ok {
		return nil, ErrGestorNaoEncontrado
	}

	copia := *gestor
	emailAnterior := copia.Email

	if err := atualizar(&copia); err != nil {
		return nil, err
	}

	if strings.TrimSpace(copia.NovaSenhaHash) != "" {
		copia.SenhaHash = strings.TrimSpace(copia.NovaSenhaHash)
		copia.NovaSenhaHash = ""
	}

	emailNovo := normalizarEmailGestor(copia.Email)
	if emailNovo != emailAnterior {
		if idExistente, ok := r.porMail[emailNovo]; ok && idExistente != id {
			return nil, ErrEmailGestorDuplicado
		}
		delete(r.porMail, emailAnterior)
		r.porMail[emailNovo] = id
		copia.Email = emailNovo
	}

	copia.AtualizadoEm = time.Now().UTC()
	*gestor = copia
	return clonarGestor(gestor), nil
}

func (r *gestorRedeMemoria) Contar() (int, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := len(r.porID)
	ativos := 0
	for _, gestor := range r.porID {
		if gestor.Ativo {
			ativos++
		}
	}
	return total, ativos, nil
}

func (r *gestorRedeMemoria) ExisteGestorPorEmail(email string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.porMail[normalizarEmailGestor(email)]
	return ok, nil
}

func (r *gestorRedeMemoria) BuscarPorEmailParaLogin(email string) (*modelos.GestorRede, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.porMail[normalizarEmailGestor(email)]
	if !ok {
		return nil, ErrGestorNaoEncontrado
	}
	gestor, ok := r.porID[id]
	if !ok {
		return nil, ErrGestorNaoEncontrado
	}
	return clonarGestor(gestor), nil
}

func normalizarEmailGestor(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func clonarGestor(orig *modelos.GestorRede) *modelos.GestorRede {
	copia := *orig
	return &copia
}
