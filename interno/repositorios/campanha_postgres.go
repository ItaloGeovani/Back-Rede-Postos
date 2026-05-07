package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

type campanhaPostgres struct {
	db *sql.DB
}

func NovoCampanhaPostgres(db *sql.DB) *campanhaPostgres {
	return &campanhaPostgres{db: db}
}

var ErrCampanhaNaoEncontrada = errors.New("campanha nao encontrada")

func (r *campanhaPostgres) resolverCriadoPor(sessaoID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessaoID = strings.TrimSpace(sessaoID)
	var id string
	err := r.db.QueryRowContext(ctx, `
SELECT id::text FROM usuarios WHERE id::text = $1 LIMIT 1`, sessaoID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	err = r.db.QueryRowContext(ctx, `
SELECT id::text FROM usuarios WHERE papel = 'super_admin' ORDER BY criado_em ASC LIMIT 1`).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("nenhum usuario super_admin no banco para registrar campanha")
		}
		return "", err
	}
	return id, nil
}

func (r *campanhaPostgres) PostoPertenceARede(idPosto, idRede string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var um int
	err := r.db.QueryRowContext(ctx, `
SELECT 1 FROM postos WHERE id = $1::uuid AND rede_id = $2::uuid LIMIT 1`, idPosto, idRede).Scan(&um)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func preencherTituloExibicaoEscopo(c *modelos.Campanha) {
	if strings.TrimSpace(c.Titulo) != "" {
		c.TituloExibicao = strings.TrimSpace(c.Titulo)
	} else {
		c.TituloExibicao = strings.TrimSpace(c.Nome)
	}
	if strings.TrimSpace(c.IDPosto) == "" {
		c.Escopo = "rede"
	} else {
		c.Escopo = "posto"
	}
}

func (r *campanhaPostgres) ListarPorRedeID(idRede string) ([]*modelos.Campanha, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const query = `
SELECT
  c.id::text,
  c.rede_id::text,
  c.nome,
  COALESCE(c.titulo, ''),
  COALESCE(c.descricao, ''),
  COALESCE(c.imagem_url, ''),
  COALESCE(c.posto_id::text, ''),
  c.status::text,
  c.vigencia_inicio,
  c.vigencia_fim,
  c.valida_no_app,
  c.valida_no_posto_fisico,
  c.tipo_beneficio,
  c.modalidade_desconto,
  c.base_desconto,
  c.valor_desconto::float8,
  c.valor_minimo_compra::float8,
  c.valor_maximo_compra::float8,
  c.max_usos_por_cliente,
  c.litros_min::float8,
  c.litros_max::float8,
  c.criado_por::text,
  c.criado_em,
  c.atualizado_em
FROM campanhas c
WHERE c.rede_id = $1::uuid
ORDER BY c.vigencia_inicio DESC NULLS LAST, c.criado_em DESC`

	rows, err := r.db.QueryContext(ctx, query, strings.TrimSpace(idRede))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []*modelos.Campanha
	var idsCampanhas []string
	for rows.Next() {
		var c modelos.Campanha
		var vigIni, vigFim sql.NullTime
		var maxUsos sql.NullInt64
		var vMaxCompra sql.NullFloat64
		var litMin, litMax sql.NullFloat64
		if err := rows.Scan(
			&c.ID, &c.IDRede, &c.Nome, &c.Titulo, &c.Descricao, &c.ImagemURL,
			&c.IDPosto, &c.Status, &vigIni, &vigFim,
			&c.ValidaNoApp, &c.ValidaNoPostoFisico,
			&c.TipoBeneficio,
			&c.ModalidadeDesconto, &c.BaseDesconto,
			&c.ValorDesconto, &c.ValorMinimoCompra, &vMaxCompra,
			&maxUsos,
			&litMin, &litMax,
			&c.CriadoPor, &c.CriadoEm, &c.AtualizadoEm,
		); err != nil {
			return nil, err
		}
		if vigIni.Valid {
			t := vigIni.Time
			c.VigenciaInicio = &t
		}
		if vigFim.Valid {
			t := vigFim.Time
			c.VigenciaFim = &t
		}
		if maxUsos.Valid {
			v := int(maxUsos.Int64)
			c.MaxUsosPorCliente = &v
		}
		if litMin.Valid {
			v := litMin.Float64
			c.LitrosMin = &v
		}
		if litMax.Valid {
			v := litMax.Float64
			c.LitrosMax = &v
		}
		if vMaxCompra.Valid {
			v := vMaxCompra.Float64
			c.ValorMaximoCompra = &v
		}
		preencherTituloExibicaoEscopo(&c)
		lista = append(lista, &c)
		idsCampanhas = append(idsCampanhas, c.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(lista) == 0 {
		return lista, nil
	}
	m, err := r.mapaCombustiveisCampanhas(ctx, idsCampanhas)
	if err != nil {
		return nil, err
	}
	for _, c := range lista {
		if ids, ok := m[c.ID]; ok {
			c.IDsCombustiveisRede = ids
		} else {
			c.IDsCombustiveisRede = nil
		}
	}
	return lista, nil
}

// BuscarPorIDeRede retorna a campanha; ErrCampanhaNaoEncontrada se o id nao for da rede.
func (r *campanhaPostgres) BuscarPorIDeRede(idCampanha, idRede string) (*modelos.Campanha, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	idCampanha = strings.TrimSpace(idCampanha)
	idRede = strings.TrimSpace(idRede)
	if idCampanha == "" || idRede == "" {
		return nil, ErrCampanhaNaoEncontrada
	}
	const q = `
SELECT
  c.id::text,
  c.rede_id::text,
  c.nome,
  COALESCE(c.titulo, ''),
  COALESCE(c.descricao, ''),
  COALESCE(c.imagem_url, ''),
  COALESCE(c.posto_id::text, ''),
  c.status::text,
  c.vigencia_inicio,
  c.vigencia_fim,
  c.valida_no_app,
  c.valida_no_posto_fisico,
  c.tipo_beneficio,
  c.modalidade_desconto,
  c.base_desconto,
  c.valor_desconto::float8,
  c.valor_minimo_compra::float8,
  c.valor_maximo_compra::float8,
  c.max_usos_por_cliente,
  c.litros_min::float8,
  c.litros_max::float8,
  c.criado_por::text,
  c.criado_em,
  c.atualizado_em
FROM campanhas c
WHERE c.id = $1::uuid AND c.rede_id = $2::uuid
LIMIT 1`
	var c modelos.Campanha
	var vigIni, vigFim sql.NullTime
	var maxUsos sql.NullInt64
	var vMaxCompra sql.NullFloat64
	var litMin, litMax sql.NullFloat64
	err := r.db.QueryRowContext(ctx, q, idCampanha, idRede).Scan(
		&c.ID, &c.IDRede, &c.Nome, &c.Titulo, &c.Descricao, &c.ImagemURL,
		&c.IDPosto, &c.Status, &vigIni, &vigFim,
		&c.ValidaNoApp, &c.ValidaNoPostoFisico,
		&c.TipoBeneficio,
		&c.ModalidadeDesconto, &c.BaseDesconto,
		&c.ValorDesconto, &c.ValorMinimoCompra, &vMaxCompra,
		&maxUsos,
		&litMin, &litMax,
		&c.CriadoPor, &c.CriadoEm, &c.AtualizadoEm,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCampanhaNaoEncontrada
	}
	if err != nil {
		return nil, err
	}
	if vigIni.Valid {
		t := vigIni.Time
		c.VigenciaInicio = &t
	}
	if vigFim.Valid {
		t := vigFim.Time
		c.VigenciaFim = &t
	}
	if maxUsos.Valid {
		v := int(maxUsos.Int64)
		c.MaxUsosPorCliente = &v
	}
	if litMin.Valid {
		v := litMin.Float64
		c.LitrosMin = &v
	}
	if litMax.Valid {
		v := litMax.Float64
		c.LitrosMax = &v
	}
	if vMaxCompra.Valid {
		v := vMaxCompra.Float64
		c.ValorMaximoCompra = &v
	}
	preencherTituloExibicaoEscopo(&c)
	m, err := r.mapaCombustiveisCampanhas(ctx, []string{c.ID})
	if err != nil {
		return nil, err
	}
	if ids, ok := m[c.ID]; ok {
		c.IDsCombustiveisRede = ids
	}
	return &c, nil
}

func (r *campanhaPostgres) mapaCombustiveisCampanhas(ctx context.Context, idsCampanha []string) (map[string][]string, error) {
	out := make(map[string][]string)
	if len(idsCampanha) == 0 {
		return out, nil
	}
	place := make([]string, len(idsCampanha))
	args := make([]any, len(idsCampanha))
	for i, id := range idsCampanha {
		place[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	q := `SELECT campanha_id::text, combustivel_rede_id::text
FROM campanha_combustiveis_rede
WHERE campanha_id IN (` + strings.Join(place, ",") + `)`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var campID, combID string
		if err := rows.Scan(&campID, &combID); err != nil {
			return nil, err
		}
		out[campID] = append(out[campID], combID)
	}
	return out, rows.Err()
}

func (r *campanhaPostgres) Criar(sessaoCriador string, c *modelos.Campanha) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	criadoPor, err := r.resolverCriadoPor(sessaoCriador)
	if err != nil {
		return err
	}

	var posto any
	if strings.TrimSpace(c.IDPosto) == "" {
		posto = nil
	} else {
		posto = strings.TrimSpace(c.IDPosto)
	}

	var vigIni, vigFim any
	if c.VigenciaInicio != nil {
		vigIni = *c.VigenciaInicio
	}
	if c.VigenciaFim != nil {
		vigFim = *c.VigenciaFim
	}

	var maxUsos any
	if c.MaxUsosPorCliente != nil {
		maxUsos = *c.MaxUsosPorCliente
	}
	var litMin, litMax any
	if c.LitrosMin != nil {
		litMin = *c.LitrosMin
	}
	if c.LitrosMax != nil {
		litMax = *c.LitrosMax
	}
	var vMax any
	if c.ValorMaximoCompra != nil {
		vMax = *c.ValorMaximoCompra
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const query = `
INSERT INTO campanhas (
  rede_id, nome, descricao, status, criado_por,
  imagem_url, titulo, posto_id, vigencia_inicio, vigencia_fim,
  valida_no_app, valida_no_posto_fisico,
  tipo_beneficio, modalidade_desconto, base_desconto, valor_desconto, valor_minimo_compra, valor_maximo_compra, max_usos_por_cliente,
  litros_min, litros_max
)
VALUES (
  $1::uuid, $2, NULLIF($3, ''), $4::status_campanha, $5::uuid,
  NULLIF($6, ''), NULLIF($7, ''), $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17, $18, $19,
  $20, $21
)
RETURNING id::text, criado_em, atualizado_em`

	err = tx.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(c.IDRede),
		strings.TrimSpace(c.Nome),
		strings.TrimSpace(c.Descricao),
		string(c.Status),
		criadoPor,
		strings.TrimSpace(c.ImagemURL),
		strings.TrimSpace(c.Titulo),
		posto,
		vigIni,
		vigFim,
		c.ValidaNoApp,
		c.ValidaNoPostoFisico,
		strings.TrimSpace(c.TipoBeneficio),
		strings.TrimSpace(c.ModalidadeDesconto),
		strings.TrimSpace(c.BaseDesconto),
		c.ValorDesconto,
		c.ValorMinimoCompra,
		vMax,
		maxUsos,
		litMin,
		litMax,
	).Scan(&c.ID, &c.CriadoEm, &c.AtualizadoEm)
	if err != nil {
		return err
	}
	if err := r.sincronizarCombustiveisTx(ctx, tx, c.ID, c.IDsCombustiveisRede); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	c.CriadoPor = criadoPor
	preencherTituloExibicaoEscopo(c)
	return nil
}

func (r *campanhaPostgres) Atualizar(c *modelos.Campanha) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var posto any
	if strings.TrimSpace(c.IDPosto) == "" {
		posto = nil
	} else {
		posto = strings.TrimSpace(c.IDPosto)
	}

	var vigIni, vigFim any
	if c.VigenciaInicio != nil {
		vigIni = *c.VigenciaInicio
	}
	if c.VigenciaFim != nil {
		vigFim = *c.VigenciaFim
	}

	var maxUsos any
	if c.MaxUsosPorCliente != nil {
		maxUsos = *c.MaxUsosPorCliente
	}
	var litMin, litMax any
	if c.LitrosMin != nil {
		litMin = *c.LitrosMin
	}
	if c.LitrosMax != nil {
		litMax = *c.LitrosMax
	}
	var vMax any
	if c.ValorMaximoCompra != nil {
		vMax = *c.ValorMaximoCompra
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const query = `
UPDATE campanhas SET
  nome = $1,
  descricao = NULLIF($2, ''),
  status = $3::status_campanha,
  imagem_url = NULLIF($4, ''),
  titulo = NULLIF($5, ''),
  posto_id = $6,
  vigencia_inicio = $7,
  vigencia_fim = $8,
  valida_no_app = $9,
  valida_no_posto_fisico = $10,
  tipo_beneficio = $11,
  modalidade_desconto = $12,
  base_desconto = $13,
  valor_desconto = $14,
  valor_minimo_compra = $15,
  valor_maximo_compra = $16,
  max_usos_por_cliente = $17,
  litros_min = $18,
  litros_max = $19,
  atualizado_em = NOW()
WHERE id = $20::uuid AND rede_id = $21::uuid`

	res, err := tx.ExecContext(
		ctx,
		query,
		strings.TrimSpace(c.Nome),
		strings.TrimSpace(c.Descricao),
		string(c.Status),
		strings.TrimSpace(c.ImagemURL),
		strings.TrimSpace(c.Titulo),
		posto,
		vigIni,
		vigFim,
		c.ValidaNoApp,
		c.ValidaNoPostoFisico,
		strings.TrimSpace(c.TipoBeneficio),
		strings.TrimSpace(c.ModalidadeDesconto),
		strings.TrimSpace(c.BaseDesconto),
		c.ValorDesconto,
		c.ValorMinimoCompra,
		vMax,
		maxUsos,
		litMin,
		litMax,
		strings.TrimSpace(c.ID),
		strings.TrimSpace(c.IDRede),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrCampanhaNaoEncontrada
	}
	if err := r.sincronizarCombustiveisTx(ctx, tx, c.ID, c.IDsCombustiveisRede); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *campanhaPostgres) sincronizarCombustiveisTx(ctx context.Context, tx *sql.Tx, campanhaID string, ids []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM campanha_combustiveis_rede WHERE campanha_id = $1::uuid`, strings.TrimSpace(campanhaID)); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO campanha_combustiveis_rede (campanha_id, combustivel_rede_id)
VALUES ($1::uuid, $2::uuid)`,
			strings.TrimSpace(campanhaID), id,
		); err != nil {
			return err
		}
	}
	return nil
}
