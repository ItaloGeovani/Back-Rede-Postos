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

type postoPostgres struct {
	db *sql.DB
}

func NovoPostoPostgres(db *sql.DB) *postoPostgres {
	return &postoPostgres{db: db}
}

var ErrCodigoPostoDuplicadoNaRede = errors.New("codigo do posto ja existe nesta rede")
var ErrCNPJPostoDuplicado = errors.New("cnpj ja cadastrado para outro posto")
var ErrPostoNaoEncontradoNaRede = errors.New("posto nao encontrado nesta rede")

func (r *postoPostgres) ListarPorRedeID(idRede string) ([]*modelos.Posto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  id::text,
  rede_id::text,
  nome,
  codigo,
  COALESCE(nome_fantasia, ''),
  COALESCE(cnpj, ''),
  COALESCE(logo_url, ''),
  COALESCE(rua, ''),
  COALESCE(numero, ''),
  COALESCE(bairro, ''),
  COALESCE(complemento, ''),
  COALESCE(cep, ''),
  COALESCE(cidade, ''),
  COALESCE(estado, ''),
  COALESCE(telefone, ''),
  COALESCE(email_contato, ''),
  criado_em,
  atualizado_em
FROM postos
WHERE rede_id = $1::uuid
ORDER BY nome ASC`

	rows, err := r.db.QueryContext(ctx, query, idRede)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []*modelos.Posto
	for rows.Next() {
		var p modelos.Posto
		if err := rows.Scan(
			&p.ID, &p.IDRede, &p.Nome, &p.Codigo,
			&p.NomeFantasia, &p.CNPJ, &p.LogoURL,
			&p.Rua, &p.Numero, &p.Bairro, &p.Complemento, &p.CEP,
			&p.Cidade, &p.Estado, &p.Telefone, &p.EmailContato,
			&p.CriadoEm, &p.AtualizadoEm,
		); err != nil {
			return nil, err
		}
		lista = append(lista, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lista, nil
}

func (r *postoPostgres) Criar(p *modelos.Posto) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
INSERT INTO postos (
  rede_id, nome, codigo,
  nome_fantasia, cnpj, logo_url,
  rua, numero, bairro, complemento, cep,
  cidade, estado, telefone, email_contato
)
VALUES (
  $1::uuid, $2, $3,
  NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
  NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''),
  NULLIF($12, ''), NULLIF($13, ''),
  NULLIF($14, ''), NULLIF($15, '')
)
RETURNING id::text, criado_em, atualizado_em`

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(p.IDRede),
		strings.TrimSpace(p.Nome),
		strings.TrimSpace(p.Codigo),
		strings.TrimSpace(p.NomeFantasia),
		strings.TrimSpace(p.CNPJ),
		strings.TrimSpace(p.LogoURL),
		strings.TrimSpace(p.Rua),
		strings.TrimSpace(p.Numero),
		strings.TrimSpace(p.Bairro),
		strings.TrimSpace(p.Complemento),
		strings.TrimSpace(p.CEP),
		strings.TrimSpace(p.Cidade),
		strings.TrimSpace(p.Estado),
		strings.TrimSpace(p.Telefone),
		strings.TrimSpace(p.EmailContato),
	).Scan(&p.ID, &p.CriadoEm, &p.AtualizadoEm)
	if err != nil {
		return mapearErroPostoPostgres(err)
	}
	return nil
}

func (r *postoPostgres) AtualizarNaRede(p *modelos.Posto) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
UPDATE postos SET
  nome = $3,
  codigo = $4,
  nome_fantasia = NULLIF($5, ''),
  cnpj = NULLIF($6, ''),
  logo_url = NULLIF($7, ''),
  rua = NULLIF($8, ''),
  numero = NULLIF($9, ''),
  bairro = NULLIF($10, ''),
  complemento = NULLIF($11, ''),
  cep = NULLIF($12, ''),
  cidade = NULLIF($13, ''),
  estado = NULLIF($14, ''),
  telefone = NULLIF($15, ''),
  email_contato = NULLIF($16, '')
WHERE id = $1::uuid AND rede_id = $2::uuid
RETURNING criado_em, atualizado_em`

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(p.ID),
		strings.TrimSpace(p.IDRede),
		strings.TrimSpace(p.Nome),
		strings.TrimSpace(p.Codigo),
		strings.TrimSpace(p.NomeFantasia),
		strings.TrimSpace(p.CNPJ),
		strings.TrimSpace(p.LogoURL),
		strings.TrimSpace(p.Rua),
		strings.TrimSpace(p.Numero),
		strings.TrimSpace(p.Bairro),
		strings.TrimSpace(p.Complemento),
		strings.TrimSpace(p.CEP),
		strings.TrimSpace(p.Cidade),
		strings.TrimSpace(p.Estado),
		strings.TrimSpace(p.Telefone),
		strings.TrimSpace(p.EmailContato),
	).Scan(&p.CriadoEm, &p.AtualizadoEm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPostoNaoEncontradoNaRede
		}
		return mapearErroPostoPostgres(err)
	}
	return nil
}

func mapearErroPostoPostgres(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code == "23505" {
		cn := strings.ToLower(pgErr.ConstraintName)
		msg := strings.ToLower(pgErr.Message)
		if strings.Contains(cn, "cnpj") || strings.Contains(msg, "uq_postos_cnpj") {
			return ErrCNPJPostoDuplicado
		}
		return ErrCodigoPostoDuplicadoNaRede
	}
	return err
}
