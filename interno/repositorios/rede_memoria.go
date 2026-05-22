package repositorios

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"gaspass-servidor/interno/modelos"
)

var (
	ErrRedeNaoEncontrada = errors.New("rede nao encontrada")
	ErrRedeCNPJDuplicado = errors.New("cnpj da rede ja cadastrado")
	ErrRedeNomeDuplicado = errors.New("nome fantasia da rede ja cadastrado")
)

type RedeRepositorio interface {
	Listar() ([]*modelos.Rede, error)
	BuscarPorID(id string) (*modelos.Rede, error)
	Criar(rede *modelos.Rede) error
	Atualizar(id string, atualizar func(*modelos.Rede) error) (*modelos.Rede, error)
}

type redeMemoria struct {
	mu      sync.RWMutex
	porID   map[string]*modelos.Rede
	porCNPJ map[string]string
	porNome map[string]string
}

func NovoRedeMemoria() RedeRepositorio {
	return &redeMemoria{
		porID:   make(map[string]*modelos.Rede),
		porCNPJ: make(map[string]string),
		porNome: make(map[string]string),
	}
}

func (r *redeMemoria) Listar() ([]*modelos.Rede, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lista := make([]*modelos.Rede, 0, len(r.porID))
	for _, item := range r.porID {
		copia := *item
		lista = append(lista, &copia)
	}

	sort.Slice(lista, func(i, j int) bool {
		return lista[i].CriadoEm.Before(lista[j].CriadoEm)
	})
	return lista, nil
}

func (r *redeMemoria) BuscarPorID(id string) (*modelos.Rede, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rede, ok := r.porID[id]
	if !ok {
		return nil, ErrRedeNaoEncontrada
	}

	copia := *rede
	return &copia, nil
}

func (r *redeMemoria) Criar(rede *modelos.Rede) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cnpj := normalizarCNPJ(rede.CNPJ)
	nome := normalizarNomeRede(rede.NomeFantasia)
	if _, ok := r.porCNPJ[cnpj]; ok {
		return ErrRedeCNPJDuplicado
	}
	if _, ok := r.porNome[nome]; ok {
		return ErrRedeNomeDuplicado
	}

	copia := *rede
	copia.CNPJ = cnpj
	copia.NomeFantasia = strings.TrimSpace(rede.NomeFantasia)
	if strings.TrimSpace(copia.MoedaVirtualNome) == "" {
		copia.MoedaVirtualNome = "Creditos"
	}
	if copia.MoedaVirtualCotacao <= 0 {
		copia.MoedaVirtualCotacao = 1
	}
	if copia.VoucherDiasValidadeResgate < 1 {
		copia.VoucherDiasValidadeResgate = 7
	}
	if copia.VoucherMinutosExpiraPagamentoPix < 5 {
		copia.VoucherMinutosExpiraPagamentoPix = 30
	}
	if strings.TrimSpace(copia.GatewayPagamentoModo) == "" {
		copia.GatewayPagamentoModo = modelos.GatewayPagamentoModoRede
	}
	if strings.TrimSpace(copia.GatewayProvedorAtivo) == "" {
		copia.GatewayProvedorAtivo = modelos.GatewayProvedorMercadoPago
	}
	if !copia.GatewayMeiosHabilitados.Pix && !copia.GatewayMeiosHabilitados.CartaoCredito && !copia.GatewayMeiosHabilitados.CartaoDebito {
		copia.GatewayMeiosHabilitados = modelos.MeiosPadrao()
	}
	now := time.Now().UTC()
	copia.CriadoEm = now
	copia.AtualizadoEm = now

	r.porID[copia.ID] = &copia
	r.porCNPJ[cnpj] = copia.ID
	r.porNome[nome] = copia.ID
	return nil
}

func (r *redeMemoria) Atualizar(id string, atualizar func(*modelos.Rede) error) (*modelos.Rede, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rede, ok := r.porID[id]
	if !ok {
		return nil, ErrRedeNaoEncontrada
	}

	copia := *rede
	cnpjAnterior := normalizarCNPJ(copia.CNPJ)
	nomeAnterior := normalizarNomeRede(copia.NomeFantasia)

	if err := atualizar(&copia); err != nil {
		return nil, err
	}

	cnpjNovo := normalizarCNPJ(copia.CNPJ)
	nomeNovo := normalizarNomeRede(copia.NomeFantasia)

	if cnpjNovo != cnpjAnterior {
		if idExistente, ok := r.porCNPJ[cnpjNovo]; ok && idExistente != id {
			return nil, ErrRedeCNPJDuplicado
		}
		delete(r.porCNPJ, cnpjAnterior)
		r.porCNPJ[cnpjNovo] = id
	}

	if nomeNovo != nomeAnterior {
		if idExistente, ok := r.porNome[nomeNovo]; ok && idExistente != id {
			return nil, ErrRedeNomeDuplicado
		}
		delete(r.porNome, nomeAnterior)
		r.porNome[nomeNovo] = id
	}

	copia.CNPJ = cnpjNovo
	copia.NomeFantasia = strings.TrimSpace(copia.NomeFantasia)
	copia.AtualizadoEm = time.Now().UTC()
	*rede = copia

	clonada := *rede
	return &clonada, nil
}

func normalizarCNPJ(cnpj string) string {
	return strings.TrimSpace(cnpj)
}

func normalizarNomeRede(nome string) string {
	return strings.ToLower(strings.TrimSpace(nome))
}
