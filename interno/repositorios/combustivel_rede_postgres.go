package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type combustivelRedePostgres struct {
	db *sql.DB
}

func NovoCombustivelRedePostgres(db *sql.DB) CombustivelRedeRepositorio {
	return &combustivelRedePostgres{db: db}
}

const selectCombustivelCols = `
  id::text, rede_id::text, posto_id::text, TRIM(nome), COALESCE(TRIM(codigo), ''), COALESCE(descricao, ''),
  preco_por_litro::float8, ativo, ordem, criado_em, atualizado_em`

func scanCombustivel(scanner interface {
	Scan(dest ...any) error
}, x *CombustivelRedeRegistro) error {
	return scanner.Scan(
		&x.ID, &x.RedeID, &x.PostoID, &x.Nome, &x.Codigo, &x.Descricao,
		&x.PrecoPorLitro, &x.Ativo, &x.Ordem, &x.CriadoEm, &x.AtualizadoEm,
	)
}

func (r *combustivelRedePostgres) ListarPorRede(redeID, postoID string) ([]*CombustivelRedeRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	redeID = strings.TrimSpace(redeID)
	postoID = strings.TrimSpace(postoID)
	var (
		rows *sql.Rows
		err  error
	)
	if postoID != "" {
		q := `
SELECT` + selectCombustivelCols + `
FROM rede_combustiveis
WHERE rede_id = $1::uuid AND posto_id = $2::uuid
ORDER BY ordem ASC, nome ASC`
		rows, err = r.db.QueryContext(ctx, q, redeID, postoID)
	} else {
		q := `
SELECT` + selectCombustivelCols + `
FROM rede_combustiveis
WHERE rede_id = $1::uuid
ORDER BY posto_id ASC, ordem ASC, nome ASC`
		rows, err = r.db.QueryContext(ctx, q, redeID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*CombustivelRedeRegistro
	for rows.Next() {
		var x CombustivelRedeRegistro
		if err := scanCombustivel(rows, &x); err != nil {
			return nil, err
		}
		out = append(out, &x)
	}
	return out, rows.Err()
}

func (r *combustivelRedePostgres) BuscarPorID(id, redeID string) (*CombustivelRedeRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	q := `
SELECT` + selectCombustivelCols + `
FROM rede_combustiveis
WHERE id = $1::uuid AND rede_id = $2::uuid`
	var x CombustivelRedeRegistro
	err := scanCombustivel(
		r.db.QueryRowContext(ctx, q, strings.TrimSpace(id), strings.TrimSpace(redeID)),
		&x,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCombustivelRedeNaoEncontrado
		}
		return nil, err
	}
	return &x, nil
}

func (r *combustivelRedePostgres) Criar(x *CombustivelRedeRegistro) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	var cod, des sql.NullString
	if t := strings.TrimSpace(x.Codigo); t != "" {
		cod = sql.NullString{String: t, Valid: true}
	}
	if t := strings.TrimSpace(x.Descricao); t != "" {
		des = sql.NullString{String: t, Valid: true}
	}
	const q = `
INSERT INTO rede_combustiveis (rede_id, posto_id, nome, codigo, descricao, preco_por_litro, ativo, ordem)
SELECT $1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8
WHERE EXISTS (
  SELECT 1 FROM postos p
  WHERE p.id = $2::uuid AND p.rede_id = $1::uuid
)
RETURNING id::text, criado_em, atualizado_em`
	err := r.db.QueryRowContext(
		ctx, q,
		strings.TrimSpace(x.RedeID),
		strings.TrimSpace(x.PostoID),
		strings.TrimSpace(x.Nome),
		cod,
		des,
		x.PrecoPorLitro,
		x.Ativo,
		x.Ordem,
	).Scan(&x.ID, &x.CriadoEm, &x.AtualizadoEm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPostoNaoPertenceARede
		}
		return mapearErrCombustivelPG(err)
	}
	return nil
}

func (r *combustivelRedePostgres) Atualizar(id, redeID string, atualizar func(*CombustivelRedeRegistro) error) (*CombustivelRedeRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	qBusca := `
SELECT` + selectCombustivelCols + `
FROM rede_combustiveis
WHERE id = $1::uuid AND rede_id = $2::uuid
FOR UPDATE`
	var row CombustivelRedeRegistro
	err = scanCombustivel(
		tx.QueryRowContext(ctx, qBusca, strings.TrimSpace(id), strings.TrimSpace(redeID)),
		&row,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCombustivelRedeNaoEncontrado
		}
		return nil, err
	}
	if err := atualizar(&row); err != nil {
		return nil, err
	}
	var codSet, desSet sql.NullString
	if t := strings.TrimSpace(row.Codigo); t != "" {
		codSet = sql.NullString{String: t, Valid: true}
	}
	if t := strings.TrimSpace(row.Descricao); t != "" {
		desSet = sql.NullString{String: t, Valid: true}
	}
	const qUp = `
UPDATE rede_combustiveis
SET
  nome = $3,
  codigo = $4,
  descricao = $5,
  preco_por_litro = $6,
  ativo = $7,
  ordem = $8,
  atualizado_em = NOW()
WHERE id = $1::uuid AND rede_id = $2::uuid
RETURNING atualizado_em`
	err = tx.QueryRowContext(
		ctx, qUp,
		strings.TrimSpace(id),
		strings.TrimSpace(redeID),
		strings.TrimSpace(row.Nome),
		codSet,
		desSet,
		row.PrecoPorLitro,
		row.Ativo,
		row.Ordem,
	).Scan(&row.AtualizadoEm)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapearErrCombustivelPG(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *combustivelRedePostgres) Excluir(id, redeID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	res, err := r.db.ExecContext(ctx, `
DELETE FROM rede_combustiveis
WHERE id = $1::uuid AND rede_id = $2::uuid`,
		strings.TrimSpace(id), strings.TrimSpace(redeID),
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrCombustivelRedeNaoEncontrado
	}
	return nil
}

func (r *combustivelRedePostgres) ExpandirIDsMesmoTipo(redeID string, ids []string) ([]string, error) {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" || len(ids) == 0 {
		return nil, nil
	}
	clean := make([]string, 0, len(ids))
	seenIn := make(map[string]struct{})
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seenIn[id]; ok {
			continue
		}
		seenIn[id] = struct{}{}
		clean = append(clean, id)
	}
	if len(clean) == 0 {
		return nil, nil
	}
	todos, err := r.ListarPorRede(redeID, "")
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*CombustivelRedeRegistro, len(todos))
	for _, c := range todos {
		if c != nil {
			byID[c.ID] = c
		}
	}
	type chave struct {
		porCodigo bool
		valor     string
	}
	chaves := make(map[chave]struct{})
	for _, id := range clean {
		c, ok := byID[id]
		if !ok {
			continue
		}
		cod := strings.ToLower(strings.TrimSpace(c.Codigo))
		if cod != "" {
			chaves[chave{porCodigo: true, valor: cod}] = struct{}{}
		} else {
			chaves[chave{porCodigo: false, valor: strings.ToLower(strings.TrimSpace(c.Nome))}] = struct{}{}
		}
	}
	out := make([]string, 0)
	seenOut := make(map[string]struct{})
	for _, c := range todos {
		if c == nil {
			continue
		}
		cod := strings.ToLower(strings.TrimSpace(c.Codigo))
		match := false
		if cod != "" {
			_, match = chaves[chave{porCodigo: true, valor: cod}]
		}
		if !match {
			_, match = chaves[chave{porCodigo: false, valor: strings.ToLower(strings.TrimSpace(c.Nome))}]
		}
		if !match {
			continue
		}
		if _, ok := seenOut[c.ID]; ok {
			continue
		}
		seenOut[c.ID] = struct{}{}
		out = append(out, c.ID)
	}
	return out, nil
}

func mapearErrCombustivelPG(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	name := strings.ToLower(pgErr.ConstraintName)
	if strings.Contains(name, "uq_rede_combustivel_posto_codigo") ||
		strings.Contains(name, "uq_rede_combustivel_codigo") {
		return errors.New("ja existe combustivel com este codigo neste posto")
	}
	return err
}
