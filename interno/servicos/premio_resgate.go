package servicos

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/utils"

	"github.com/google/uuid"
)

const (
	tipoRefResgatePremio        = "resgate_premio"
	tipoRefResgatePremioEstorno = "resgate_premio_estorno"
	diasUteisRetiradaPremio     = 2
)

var (
	ErrPremioIndisponivel     = errors.New("premio indisponivel para resgate")
	ErrPremioResgateSaldo     = repositorios.ErrSaldoInsuficiente
	ErrPremioResgateCancelado = errors.New("somente gestor pode cancelar resgates")
)

type premioBuscaRepo interface {
	BuscarPorIDNaRede(id, idRede string) (*modelos.Premio, error)
	DecrementarEstoqueTx(ctx context.Context, tx *sql.Tx, id, idRede string) error
	IncrementarEstoqueTx(ctx context.Context, tx *sql.Tx, id, idRede string) error
}

type ServicoPremioResgate struct {
	db      *sql.DB
	premios premioBuscaRepo
	resgates repositorios.PremioResgateRepositorio
	carteira repositorios.CarteiraRepositorio
	rede    repositorios.RedeRepositorio
}

func NovoServicoPremioResgate(
	db *sql.DB,
	premios premioBuscaRepo,
	resgates repositorios.PremioResgateRepositorio,
	carteira repositorios.CarteiraRepositorio,
	rede repositorios.RedeRepositorio,
) *ServicoPremioResgate {
	return &ServicoPremioResgate{
		db:       db,
		premios:  premios,
		resgates: resgates,
		carteira: carteira,
		rede:     rede,
	}
}

func premioResgatavelAgora(p *modelos.Premio, agora time.Time) bool {
	if p == nil || !p.Ativo {
		return false
	}
	if agora.Before(p.VigenciaInicio) {
		return false
	}
	if p.VigenciaFim != nil && agora.After(*p.VigenciaFim) {
		return false
	}
	if p.QuantidadeDisponivel != nil && *p.QuantidadeDisponivel <= 0 {
		return false
	}
	return true
}

func (s *ServicoPremioResgate) Resgatar(idRede, usuarioID, premioID string) (*modelos.PremioResgate, error) {
	idRede = strings.TrimSpace(idRede)
	usuarioID = strings.TrimSpace(usuarioID)
	premioID = strings.TrimSpace(premioID)
	if idRede == "" || usuarioID == "" || premioID == "" {
		return nil, ErrDadosInvalidos
	}
	rede, err := s.rede.BuscarPorID(idRede)
	if err != nil {
		return nil, err
	}
	if rede == nil || !rede.Ativa {
		return nil, ErrDadosInvalidos
	}
	p, err := s.premios.BuscarPorIDNaRede(premioID, idRede)
	if err != nil {
		return nil, err
	}
	agora := time.Now()
	if !premioResgatavelAgora(p, agora) {
		return nil, ErrPremioIndisponivel
	}
	saldo, err := s.carteira.ObterSaldoToken(idRede, usuarioID)
	if err != nil {
		return nil, err
	}
	if saldo+1e-9 < p.ValorMoeda {
		return nil, repositorios.ErrSaldoInsuficiente
	}
	nomeMoeda := strings.TrimSpace(rede.MoedaVirtualNome)
	if nomeMoeda == "" {
		nomeMoeda = "Luceninhas"
	}
	cotacao := rede.MoedaVirtualCotacao
	if cotacao <= 0 {
		cotacao = 1
	}
	if _, err := s.carteira.ObterOuCriarCarteira(idRede, usuarioID, nomeMoeda, cotacao); err != nil {
		return nil, err
	}

	resgateID := uuid.NewString()
	out := &modelos.PremioResgate{
		ID:                resgateID,
		IDRede:            idRede,
		PremioID:          p.ID,
		UsuarioID:         usuarioID,
		TituloSnapshot:    p.Titulo,
		ImagemURLSnapshot: p.ImagemURL,
		ValorMoeda:        p.ValorMoeda,
		Status:            modelos.PremioResgateAguardandoRetirada,
		PrazoRetiradaEm:   utils.SomarDiasUteis(agora, diasUteisRetiradaPremio),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.premios.DecrementarEstoqueTx(ctx, tx, p.ID, idRede); err != nil {
		return nil, err
	}
	if err := s.resgates.CriarTx(ctx, tx, out); err != nil {
		return nil, err
	}
	if err := s.carteira.DebitarMoedaTx(ctx, tx, idRede, usuarioID, p.ValorMoeda, tipoRefResgatePremio, resgateID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.resgates.BuscarPorIDNaRede(resgateID, idRede)
}

func (s *ServicoPremioResgate) ListarMeus(idRede, usuarioID string) ([]*modelos.PremioResgate, error) {
	if strings.TrimSpace(idRede) == "" || strings.TrimSpace(usuarioID) == "" {
		return nil, ErrDadosInvalidos
	}
	return s.resgates.ListarPorUsuario(idRede, usuarioID)
}

func (s *ServicoPremioResgate) ListarRede(idRede, status string, limite, offset int) ([]*modelos.PremioResgate, int, error) {
	if strings.TrimSpace(idRede) == "" {
		return nil, 0, ErrDadosInvalidos
	}
	status = strings.TrimSpace(status)
	if status != "" {
		switch status {
		case modelos.PremioResgateAguardandoRetirada, modelos.PremioResgateEntregue, modelos.PremioResgateCancelado:
		default:
			return nil, 0, ErrDadosInvalidos
		}
	}
	return s.resgates.ListarPorRede(idRede, status, limite, offset)
}

func (s *ServicoPremioResgate) Entregar(idRede, resgateID string, postoID *string, operador *modelos.UsuarioSessao) (*modelos.PremioResgate, error) {
	idRede = strings.TrimSpace(idRede)
	resgateID = strings.TrimSpace(resgateID)
	if idRede == "" || resgateID == "" || operador == nil {
		return nil, ErrDadosInvalidos
	}
	var postoPtr *string
	if postoID != nil && strings.TrimSpace(*postoID) != "" {
		s := strings.TrimSpace(*postoID)
		postoPtr = &s
	} else if strings.TrimSpace(operador.IDPosto) != "" {
		s := strings.TrimSpace(operador.IDPosto)
		postoPtr = &s
	}
	if err := s.resgates.MarcarEntregue(
		resgateID, idRede, postoPtr,
		operador.IDUsuario, string(operador.Papel), strings.TrimSpace(operador.NomeCompleto),
	); err != nil {
		return nil, err
	}
	return s.resgates.BuscarPorIDNaRede(resgateID, idRede)
}

func (s *ServicoPremioResgate) Cancelar(idRede, resgateID, motivo string, operador *modelos.UsuarioSessao) (*modelos.PremioResgate, error) {
	idRede = strings.TrimSpace(idRede)
	resgateID = strings.TrimSpace(resgateID)
	if idRede == "" || resgateID == "" || operador == nil {
		return nil, ErrDadosInvalidos
	}
	switch operador.Papel {
	case modelos.PapelGestorRede, modelos.PapelGerentePosto, modelos.PapelSuperAdmin:
	default:
		return nil, ErrPremioResgateCancelado
	}
	cur, err := s.resgates.BuscarPorIDNaRede(resgateID, idRede)
	if err != nil {
		return nil, err
	}
	if cur.Status != modelos.PremioResgateAguardandoRetirada {
		return nil, repositorios.ErrPremioResgateStatus
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// Marca cancelado fora da tx de crédito — usa UPDATE atômico; se falhar, rollback crédito.
	// Simplificação: cancelar no repo e depois creditar na mesma tx via raw update.
	const upd = `
UPDATE premio_resgates SET
  status = 'CANCELADO',
  cancelado_em = NOW(),
  motivo_cancelamento = NULLIF($3, '')
WHERE id = $1::uuid AND rede_id = $2::uuid AND status = 'AGUARDANDO_RETIRADA'`
	res, err := tx.ExecContext(ctx, upd, resgateID, idRede, strings.TrimSpace(motivo))
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, repositorios.ErrPremioResgateStatus
	}
	if err := s.premios.IncrementarEstoqueTx(ctx, tx, cur.PremioID, idRede); err != nil {
		return nil, err
	}
	carteiraID, err := s.carteira.ObterOuCriarCarteira(idRede, cur.UsuarioID, "Luceninhas", 1)
	if err != nil {
		return nil, err
	}
	if err := s.carteira.CreditarBonusTx(ctx, tx, idRede, carteiraID, cur.ValorMoeda, tipoRefResgatePremioEstorno, resgateID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.resgates.BuscarPorIDNaRede(resgateID, idRede)
}
