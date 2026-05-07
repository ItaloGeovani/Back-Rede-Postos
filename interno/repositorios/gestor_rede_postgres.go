package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"github.com/jackc/pgx/v5/pgconn"
)

type gestorRedePostgres struct {
	db *sql.DB
}

func NovoGestorRedePostgres(db *sql.DB) GestorRedeRepositorio {
	return &gestorRedePostgres{db: db}
}

func (r *gestorRedePostgres) Listar() ([]*modelos.GestorRede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  u.id::text,
  u.rede_id::text,
  u.nome_completo,
  u.email,
  COALESCE(u.telefone, ''),
  u.ativo,
  u.criado_em,
  u.atualizado_em,
  COALESCE(r.valor_implantacao, 0),
  COALESCE(r.valor_mensalidade, 0),
  r.primeiro_cobranca,
  COALESCE(r.dia_cobranca, 0)
FROM usuarios u
LEFT JOIN redes r ON r.id = u.rede_id
WHERE u.papel = 'gestor_rede'
ORDER BY u.criado_em DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []*modelos.GestorRede
	for rows.Next() {
		gestor, err := scanGestor(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, gestor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

func (r *gestorRedePostgres) Criar(gestor *modelos.GestorRede) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
INSERT INTO usuarios (
  rede_id,
  papel,
  nome_completo,
  email,
  senha_hash,
  ativo,
  telefone
)
VALUES ($1, 'gestor_rede', $2, $3, $4, $5, NULLIF($6, ''))
RETURNING id::text, criado_em, atualizado_em`

	err := r.db.QueryRowContext(
		ctx,
		query,
		gestor.IDRede,
		strings.TrimSpace(gestor.Nome),
		strings.TrimSpace(gestor.Email),
		gestor.SenhaHash,
		gestor.Ativo,
		strings.TrimSpace(gestor.Telefone),
	).Scan(&gestor.ID, &gestor.CriadoEm, &gestor.AtualizadoEm)
	if err != nil {
		return mapearErroGestorPostgres(err)
	}
	return nil
}

func (r *gestorRedePostgres) Atualizar(id string, atualizar func(*modelos.GestorRede) error) (*modelos.GestorRede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const queryBusca = `
SELECT
  u.id::text,
  u.rede_id::text,
  u.nome_completo,
  u.email,
  COALESCE(u.telefone, ''),
  u.ativo,
  u.criado_em,
  u.atualizado_em,
  COALESCE(r.valor_implantacao, 0),
  COALESCE(r.valor_mensalidade, 0),
  r.primeiro_cobranca,
  COALESCE(r.dia_cobranca, 0)
FROM usuarios u
LEFT JOIN redes r ON r.id = u.rede_id
WHERE u.id = $1 AND u.papel = 'gestor_rede'
FOR UPDATE OF u`

	gestor, err := scanGestor(tx.QueryRowContext(ctx, queryBusca, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGestorNaoEncontrado
		}
		return nil, err
	}

	if err := atualizar(gestor); err != nil {
		return nil, err
	}

	const queryAtualiza = `
UPDATE usuarios
SET
  nome_completo = $2,
  email = $3,
  telefone = NULLIF($4, ''),
  ativo = $5,
  senha_hash = CASE
    WHEN NULLIF(BTRIM($6::text), '') IS NOT NULL THEN BTRIM($6::text)
    ELSE senha_hash
  END,
  atualizado_em = NOW()
WHERE id = $1 AND papel = 'gestor_rede'
RETURNING atualizado_em`

	senhaOuVazio := strings.TrimSpace(gestor.NovaSenhaHash)

	err = tx.QueryRowContext(
		ctx,
		queryAtualiza,
		id,
		strings.TrimSpace(gestor.Nome),
		strings.TrimSpace(gestor.Email),
		strings.TrimSpace(gestor.Telefone),
		gestor.Ativo,
		senhaOuVazio,
	).Scan(&gestor.AtualizadoEm)
	if err != nil {
		return nil, mapearErroGestorPostgres(err)
	}

	gestor.NovaSenhaHash = ""

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return gestor, nil
}

func (r *gestorRedePostgres) BuscarPorEmailParaLogin(email string) (*modelos.GestorRede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  u.id::text,
  u.rede_id::text,
  u.nome_completo,
  u.email,
  u.senha_hash,
  COALESCE(u.telefone, ''),
  u.ativo,
  u.criado_em,
  u.atualizado_em,
  COALESCE(r.valor_implantacao, 0),
  COALESCE(r.valor_mensalidade, 0),
  r.primeiro_cobranca,
  COALESCE(r.dia_cobranca, 0)
FROM usuarios u
LEFT JOIN redes r ON r.id = u.rede_id
WHERE u.papel = 'gestor_rede'
  AND LOWER(TRIM(u.email)) = LOWER(TRIM($1))
LIMIT 1`

	gestor, err := scanGestorComSenha(r.db.QueryRowContext(ctx, query, email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGestorNaoEncontrado
		}
		return nil, err
	}
	return gestor, nil
}

func (r *gestorRedePostgres) ExisteGestorPorEmail(email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	email = strings.TrimSpace(email)
	if email == "" {
		return false, nil
	}

	const query = `
SELECT 1
FROM usuarios
WHERE papel = 'gestor_rede'
  AND LOWER(TRIM(email)) = LOWER(TRIM($1))
LIMIT 1`

	var um int
	err := r.db.QueryRowContext(ctx, query, email).Scan(&um)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *gestorRedePostgres) Contar() (int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  COUNT(*)::int AS total,
  COUNT(*) FILTER (WHERE ativo = true)::int AS ativos
FROM usuarios
WHERE papel = 'gestor_rede'`

	var total, ativos int
	if err := r.db.QueryRowContext(ctx, query).Scan(&total, &ativos); err != nil {
		return 0, 0, err
	}
	return total, ativos, nil
}

type scannerGestor interface {
	Scan(dest ...any) error
}

func scanGestor(s scannerGestor) (*modelos.GestorRede, error) {
	var gestor modelos.GestorRede
	var primeiro sql.NullTime
	err := s.Scan(
		&gestor.ID,
		&gestor.IDRede,
		&gestor.Nome,
		&gestor.Email,
		&gestor.Telefone,
		&gestor.Ativo,
		&gestor.CriadoEm,
		&gestor.AtualizadoEm,
		&gestor.ValorImplantacao,
		&gestor.ValorMensalidade,
		&primeiro,
		&gestor.DiaVencimento,
	)
	if err != nil {
		return nil, err
	}
	if primeiro.Valid {
		gestor.PrimeiroVencimento = primeiro.Time
	}
	return &gestor, nil
}

func scanGestorComSenha(s scannerGestor) (*modelos.GestorRede, error) {
	var gestor modelos.GestorRede
	var primeiro sql.NullTime
	err := s.Scan(
		&gestor.ID,
		&gestor.IDRede,
		&gestor.Nome,
		&gestor.Email,
		&gestor.SenhaHash,
		&gestor.Telefone,
		&gestor.Ativo,
		&gestor.CriadoEm,
		&gestor.AtualizadoEm,
		&gestor.ValorImplantacao,
		&gestor.ValorMensalidade,
		&primeiro,
		&gestor.DiaVencimento,
	)
	if err != nil {
		return nil, err
	}
	if primeiro.Valid {
		gestor.PrimeiroVencimento = primeiro.Time
	}
	return &gestor, nil
}

func mapearErroGestorPostgres(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code == "23505" {
		return ErrEmailGestorDuplicado
	}
	return err
}
