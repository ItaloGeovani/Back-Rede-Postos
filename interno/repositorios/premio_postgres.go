package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

type premioPostgres struct {
	db *sql.DB
}

func NovoPremioPostgres(db *sql.DB) *premioPostgres {
	return &premioPostgres{db: db}
}

var ErrPremioNaoEncontrado = errors.New("premio nao encontrado")

func (r *premioPostgres) ListarPorRedeID(idRede string) ([]*modelos.Premio, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
SELECT
  p.id::text,
  p.rede_id::text,
  p.titulo,
  COALESCE(p.imagem_url, ''),
  p.valor_moeda::float8,
  p.ativo,
  p.vigencia_inicio,
  p.vigencia_fim,
  p.quantidade_disponivel,
  p.criado_em,
  p.atualizado_em
FROM premios p
WHERE p.rede_id = $1::uuid
ORDER BY p.vigencia_inicio DESC, p.criado_em DESC`

	rows, err := r.db.QueryContext(ctx, query, strings.TrimSpace(idRede))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []*modelos.Premio
	for rows.Next() {
		var p modelos.Premio
		var vigFim sql.NullTime
		var qtd sql.NullInt64
		if err := rows.Scan(
			&p.ID, &p.IDRede, &p.Titulo, &p.ImagemURL, &p.ValorMoeda, &p.Ativo,
			&p.VigenciaInicio, &vigFim, &qtd,
			&p.CriadoEm, &p.AtualizadoEm,
		); err != nil {
			return nil, err
		}
		if vigFim.Valid {
			t := vigFim.Time
			p.VigenciaFim = &t
		}
		if qtd.Valid {
			v := int(qtd.Int64)
			p.QuantidadeDisponivel = &v
		}
		lista = append(lista, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

func (r *premioPostgres) Criar(p *modelos.Premio) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var vigFim any
	if p.VigenciaFim != nil {
		vigFim = *p.VigenciaFim
	}
	var qtd any
	if p.QuantidadeDisponivel != nil {
		qtd = *p.QuantidadeDisponivel
	}

	const query = `
INSERT INTO premios (
  rede_id, titulo, imagem_url, valor_moeda, ativo,
  vigencia_inicio, vigencia_fim, quantidade_disponivel
)
VALUES (
  $1::uuid, $2, NULLIF($3, ''), $4, $5,
  $6, $7, $8
)
RETURNING id::text, criado_em, atualizado_em`

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(p.IDRede),
		strings.TrimSpace(p.Titulo),
		strings.TrimSpace(p.ImagemURL),
		p.ValorMoeda,
		p.Ativo,
		p.VigenciaInicio,
		vigFim,
		qtd,
	).Scan(&p.ID, &p.CriadoEm, &p.AtualizadoEm)
	return err
}

func (r *premioPostgres) Atualizar(p *modelos.Premio) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var vigFim any
	if p.VigenciaFim != nil {
		vigFim = *p.VigenciaFim
	}
	var qtd any
	if p.QuantidadeDisponivel != nil {
		qtd = *p.QuantidadeDisponivel
	}

	const query = `
UPDATE premios SET
  titulo = $1,
  imagem_url = NULLIF($2, ''),
  valor_moeda = $3,
  ativo = $4,
  vigencia_inicio = $5,
  vigencia_fim = $6,
  quantidade_disponivel = $7,
  atualizado_em = NOW()
WHERE id = $8::uuid AND rede_id = $9::uuid`

	res, err := r.db.ExecContext(
		ctx,
		query,
		strings.TrimSpace(p.Titulo),
		strings.TrimSpace(p.ImagemURL),
		p.ValorMoeda,
		p.Ativo,
		p.VigenciaInicio,
		vigFim,
		qtd,
		strings.TrimSpace(p.ID),
		strings.TrimSpace(p.IDRede),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPremioNaoEncontrado
	}
	return nil
}

func (r *premioPostgres) BuscarPorIDNaRede(id, idRede string) (*modelos.Premio, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  p.id::text,
  p.rede_id::text,
  p.titulo,
  COALESCE(p.imagem_url, ''),
  p.valor_moeda::float8,
  p.ativo,
  p.vigencia_inicio,
  p.vigencia_fim,
  p.quantidade_disponivel,
  p.criado_em,
  p.atualizado_em
FROM premios p
WHERE p.id = $1::uuid AND p.rede_id = $2::uuid`

	var p modelos.Premio
	var vigFim sql.NullTime
	var qtd sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, strings.TrimSpace(id), strings.TrimSpace(idRede)).Scan(
		&p.ID, &p.IDRede, &p.Titulo, &p.ImagemURL, &p.ValorMoeda, &p.Ativo,
		&p.VigenciaInicio, &vigFim, &qtd,
		&p.CriadoEm, &p.AtualizadoEm,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPremioNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	if vigFim.Valid {
		t := vigFim.Time
		p.VigenciaFim = &t
	}
	if qtd.Valid {
		v := int(qtd.Int64)
		p.QuantidadeDisponivel = &v
	}
	return &p, nil
}

// DecrementarEstoqueTx decrementa quantidade_disponivel se controlada; NULL = ilimitado (noop).
func (r *premioPostgres) DecrementarEstoqueTx(ctx context.Context, tx *sql.Tx, id, idRede string) error {
	const check = `
SELECT quantidade_disponivel
FROM premios
WHERE id = $1::uuid AND rede_id = $2::uuid
FOR UPDATE`
	var qtd sql.NullInt64
	err := tx.QueryRowContext(ctx, check, strings.TrimSpace(id), strings.TrimSpace(idRede)).Scan(&qtd)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPremioNaoEncontrado
	}
	if err != nil {
		return err
	}
	if !qtd.Valid {
		return nil
	}
	if qtd.Int64 <= 0 {
		return ErrPremioEsgotado
	}
	const upd = `
UPDATE premios
SET quantidade_disponivel = quantidade_disponivel - 1,
    atualizado_em = NOW()
WHERE id = $1::uuid AND rede_id = $2::uuid`
	_, err = tx.ExecContext(ctx, upd, strings.TrimSpace(id), strings.TrimSpace(idRede))
	return err
}

// IncrementarEstoqueTx repõe 1 unidade se o prêmio controla estoque.
func (r *premioPostgres) IncrementarEstoqueTx(ctx context.Context, tx *sql.Tx, id, idRede string) error {
	const q = `
UPDATE premios
SET quantidade_disponivel = quantidade_disponivel + 1,
    atualizado_em = NOW()
WHERE id = $1::uuid
  AND rede_id = $2::uuid
  AND quantidade_disponivel IS NOT NULL`
	_, err := tx.ExecContext(ctx, q, strings.TrimSpace(id), strings.TrimSpace(idRede))
	return err
}

var ErrPremioEsgotado = errors.New("premio sem estoque disponivel")

