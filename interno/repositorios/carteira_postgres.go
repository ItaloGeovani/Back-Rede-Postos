package repositorios

// Modelo de razão (transacoes_carteira): o saldo é sempre SUM(valor_token * direcao).
// Lotes (carteira_lotes): cada crédito CASHBACK/BONUS cria um lote; débitos consomem FIFO
// (vence antes primeiro); expiração gera tipo EXPIRACAO no razão e zera valor_restante.

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ErrSaldoInsuficiente quando o cliente não tem token suficiente para o débito pedido.
var ErrSaldoInsuficiente = errors.New("saldo em token insuficiente")

// CarteiraLoteAtivo lote com saldo restante (para listagem no app).
type CarteiraLoteAtivo struct {
	ID                string     `json:"id"`
	ValorRestante     float64    `json:"valor_restante"`
	ValorInicial      float64    `json:"valor_inicial"`
	OrigemTipo        string     `json:"origem_tipo"`
	CreditadoEm       time.Time  `json:"creditado_em"`
	ExpiraEm          *time.Time `json:"expira_em,omitempty"`
	SegundosRestantes *int64     `json:"segundos_restantes,omitempty"`
}

type CarteiraRepositorio interface {
	ObterOuCriarCarteira(redeID, usuarioID, nomeToken string, cotacao float64) (string, error)
	CreditarBonus(
		redeID, carteiraID string,
		valorToken float64,
		tipoRef, idRef string,
	) error
	CreditarCashback(
		redeID, carteiraID string,
		valorFiat, valorToken float64,
		tipoRef, idRef string,
	) error
	CreditarBonusTx(ctx context.Context, tx *sql.Tx, redeID, carteiraID string, valorToken float64, tipoRef, idRef string) error
	CreditarCashbackTx(ctx context.Context, tx *sql.Tx, redeID, carteiraID string, valorFiat, valorToken float64, tipoRef, idRef string) error
	ObterSaldoToken(redeID, usuarioID string) (float64, error)
	DebitarMoeda(redeID, usuarioID string, valorToken float64, tipoReferencia, referenciaID string) error
	DebitarMoedaTx(ctx context.Context, tx *sql.Tx, redeID, usuarioID string, valorToken float64, tipoReferencia, referenciaID string) error
	// ExpirarLotesVencidos zera lotes vencidos e registra EXPIRACAO no razão (lazy).
	ExpirarLotesVencidos(redeID, usuarioID string) error
	// ListarLotesAtivos lista lotes com valor_restante > 0 (após lazy expire).
	ListarLotesAtivos(redeID, usuarioID string) ([]CarteiraLoteAtivo, error)
}

type carteiraPostgres struct {
	db *sql.DB
}

type dbExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func NovoCarteiraPostgres(db *sql.DB) CarteiraRepositorio {
	return &carteiraPostgres{db: db}
}

func (r *carteiraPostgres) ObterOuCriarCarteira(redeID, usuarioID, nomeToken string, cotacao float64) (string, error) {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return "", errors.New("ids invalidos")
	}
	nt := strings.TrimSpace(nomeToken)
	if nt == "" {
		nt = "Moeda"
	}
	if cotacao <= 0 {
		return "", errors.New("cotacao invalida")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	const find = `SELECT id::text FROM carteiras WHERE rede_id = $1::uuid AND usuario_id = $2::uuid`
	var id string
	err := r.db.QueryRowContext(ctx, find, redeID, usuarioID).Scan(&id)
	if err == nil && id != "" {
		return id, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	const ins = `
INSERT INTO carteiras (rede_id, usuario_id, codigo_moeda, nome_token, cotacao_token)
VALUES ($1::uuid, $2::uuid, 'BRL', $3, $4)
ON CONFLICT (rede_id, usuario_id) DO UPDATE SET
  atualizado_em = NOW()
RETURNING id::text`
	err = r.db.QueryRowContext(ctx, ins, redeID, usuarioID, nt, cotacao).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *carteiraPostgres) expiraDiasRede(ctx context.Context, q dbExec, redeID string) int {
	var d sql.NullInt64
	err := q.QueryRowContext(ctx, `
SELECT moeda_virtual_expira_dias FROM redes WHERE id = $1::uuid`, redeID).Scan(&d)
	if err != nil || !d.Valid || d.Int64 <= 0 {
		return 0
	}
	if d.Int64 > 365 {
		return 365
	}
	return int(d.Int64)
}

func (r *carteiraPostgres) criarLoteCredito(
	ctx context.Context,
	q dbExec,
	redeID, carteiraID, transacaoID, origemTipo, tipoRef, idRef string,
	valorToken float64,
	expiraDias int,
) error {
	if valorToken <= 0 || strings.TrimSpace(transacaoID) == "" {
		return nil
	}
	var existe string
	err := q.QueryRowContext(ctx, `
SELECT id::text FROM carteira_lotes WHERE transacao_id = $1::uuid LIMIT 1`, transacaoID).Scan(&existe)
	if err == nil && existe != "" {
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	var expira any
	if expiraDias > 0 {
		expira = time.Now().UTC().Add(time.Duration(expiraDias) * 24 * time.Hour)
	} else {
		expira = nil
	}
	var refID any
	if strings.TrimSpace(idRef) != "" {
		refID = strings.TrimSpace(idRef)
	} else {
		refID = nil
	}
	_, err = q.ExecContext(ctx, `
INSERT INTO carteira_lotes (
  rede_id, carteira_id, transacao_id, valor_inicial, valor_restante,
  origem_tipo, tipo_referencia, referencia_id, creditado_em, expira_em
) VALUES (
  $1::uuid, $2::uuid, $3::uuid, $4::numeric, $4::numeric,
  $5, $6, $7::uuid, NOW(), $8
)`, redeID, carteiraID, transacaoID, valorToken, origemTipo, tipoRef, refID, expira)
	return err
}

func (r *carteiraPostgres) resolverTransacaoID(
	ctx context.Context,
	q dbExec,
	redeID, tipo, tipoRef, idRef string,
) (string, error) {
	var id string
	err := q.QueryRowContext(ctx, `
SELECT id::text FROM transacoes_carteira
WHERE rede_id = $1::uuid AND tipo = $2::tipo_transacao_carteira
  AND tipo_referencia = $3 AND referencia_id = $4::uuid
LIMIT 1`, redeID, tipo, tipoRef, idRef).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *carteiraPostgres) CreditarBonus(redeID, carteiraID string, valorToken float64, tipoRef, idRef string) error {
	if valorToken <= 0 {
		return nil
	}
	redeID = strings.TrimSpace(redeID)
	carteiraID = strings.TrimSpace(carteiraID)
	tipoRef = strings.TrimSpace(tipoRef)
	idRef = strings.TrimSpace(idRef)
	if redeID == "" || carteiraID == "" || tipoRef == "" || idRef == "" {
		return errors.New("dados invalidos para bonus")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := r.creditarBonusComLote(ctx, tx, redeID, carteiraID, valorToken, tipoRef, idRef); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *carteiraPostgres) creditarBonusComLote(
	ctx context.Context, q dbExec, redeID, carteiraID string, valorToken float64, tipoRef, idRef string,
) error {
	const ins = `
INSERT INTO transacoes_carteira (
  rede_id, carteira_id, tipo, valor_fiat, valor_token, direcao, tipo_referencia, referencia_id, metadados, ocorrido_em
) VALUES (
  $1::uuid, $2::uuid, 'BONUS'::tipo_transacao_carteira, 0, $3::numeric, 1, $4, $5::uuid, '{}'::jsonb, NOW()
) ON CONFLICT (rede_id, tipo_referencia, referencia_id, tipo) DO NOTHING
RETURNING id::text`
	var txID string
	err := q.QueryRowContext(ctx, ins, redeID, carteiraID, valorToken, tipoRef, idRef).Scan(&txID)
	if errors.Is(err, sql.ErrNoRows) {
		txID, err = r.resolverTransacaoID(ctx, q, redeID, "BONUS", tipoRef, idRef)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	dias := r.expiraDiasRede(ctx, q, redeID)
	return r.criarLoteCredito(ctx, q, redeID, carteiraID, txID, "BONUS", tipoRef, idRef, valorToken, dias)
}

func (r *carteiraPostgres) CreditarCashback(redeID, carteiraID string, valorFiat, valorToken float64, tipoRef, idRef string) error {
	if valorFiat <= 0 || valorToken <= 0 {
		return nil
	}
	redeID = strings.TrimSpace(redeID)
	carteiraID = strings.TrimSpace(carteiraID)
	tipoRef = strings.TrimSpace(tipoRef)
	idRef = strings.TrimSpace(idRef)
	if redeID == "" || carteiraID == "" || tipoRef == "" || idRef == "" {
		return errors.New("dados invalidos para cashback")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := r.creditarCashbackComLote(ctx, tx, redeID, carteiraID, valorFiat, valorToken, tipoRef, idRef); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *carteiraPostgres) creditarCashbackComLote(
	ctx context.Context, q dbExec, redeID, carteiraID string, valorFiat, valorToken float64, tipoRef, idRef string,
) error {
	const ins = `
INSERT INTO transacoes_carteira (
  rede_id, carteira_id, tipo, valor_fiat, valor_token, direcao, tipo_referencia, referencia_id, metadados, ocorrido_em
) VALUES (
  $1::uuid, $2::uuid, 'CASHBACK'::tipo_transacao_carteira, $3::numeric, $4::numeric, 1, $5, $6::uuid, '{}'::jsonb, NOW()
) ON CONFLICT (rede_id, tipo_referencia, referencia_id, tipo) DO NOTHING
RETURNING id::text`
	var txID string
	err := q.QueryRowContext(ctx, ins, redeID, carteiraID, valorFiat, valorToken, tipoRef, idRef).Scan(&txID)
	if errors.Is(err, sql.ErrNoRows) {
		txID, err = r.resolverTransacaoID(ctx, q, redeID, "CASHBACK", tipoRef, idRef)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	dias := r.expiraDiasRede(ctx, q, redeID)
	return r.criarLoteCredito(ctx, q, redeID, carteiraID, txID, "CASHBACK", tipoRef, idRef, valorToken, dias)
}

func (r *carteiraPostgres) CreditarBonusTx(ctx context.Context, tx *sql.Tx, redeID, carteiraID string, valorToken float64, tipoRef, idRef string) error {
	if valorToken <= 0 {
		return nil
	}
	redeID = strings.TrimSpace(redeID)
	carteiraID = strings.TrimSpace(carteiraID)
	tipoRef = strings.TrimSpace(tipoRef)
	idRef = strings.TrimSpace(idRef)
	if redeID == "" || carteiraID == "" || tipoRef == "" || idRef == "" {
		return errors.New("dados invalidos para bonus")
	}
	return r.creditarBonusComLote(ctx, tx, redeID, carteiraID, valorToken, tipoRef, idRef)
}

func (r *carteiraPostgres) CreditarCashbackTx(ctx context.Context, tx *sql.Tx, redeID, carteiraID string, valorFiat, valorToken float64, tipoRef, idRef string) error {
	if valorFiat <= 0 || valorToken <= 0 {
		return nil
	}
	redeID = strings.TrimSpace(redeID)
	carteiraID = strings.TrimSpace(carteiraID)
	tipoRef = strings.TrimSpace(tipoRef)
	idRef = strings.TrimSpace(idRef)
	if redeID == "" || carteiraID == "" || tipoRef == "" || idRef == "" {
		return errors.New("dados invalidos para cashback")
	}
	return r.creditarCashbackComLote(ctx, tx, redeID, carteiraID, valorFiat, valorToken, tipoRef, idRef)
}

func (r *carteiraPostgres) consumirLotesFIFO(ctx context.Context, q dbExec, redeID, carteiraID string, valorToken float64) error {
	rows, err := q.QueryContext(ctx, `
SELECT id::text, valor_restante::float8
FROM carteira_lotes
WHERE rede_id = $1::uuid AND carteira_id = $2::uuid AND valor_restante > 0
ORDER BY expira_em ASC NULLS LAST, creditado_em ASC, id ASC
FOR UPDATE`, redeID, carteiraID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type loteRow struct {
		id  string
		val float64
	}
	var lotes []loteRow
	for rows.Next() {
		var L loteRow
		if err := rows.Scan(&L.id, &L.val); err != nil {
			return err
		}
		lotes = append(lotes, L)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	restante := valorToken
	for _, L := range lotes {
		if restante <= 1e-12 {
			break
		}
		take := L.val
		if take > restante {
			take = restante
		}
		_, err := q.ExecContext(ctx, `
UPDATE carteira_lotes
SET valor_restante = valor_restante - $2::numeric, atualizado_em = NOW()
WHERE id = $1::uuid AND valor_restante + 1e-12 >= $2::numeric`, L.id, take)
		if err != nil {
			return err
		}
		restante -= take
	}
	if restante > 1e-6 {
		// Sem lotes suficientes (ex.: só razão legado sem lote) — saldo do razão já validado.
		return nil
	}
	return nil
}

func (r *carteiraPostgres) debitarMoedaInterno(
	ctx context.Context, q dbExec, redeID, usuarioID string, valorToken float64, tipoReferencia, referenciaID string,
) error {
	const lockCarteira = `
SELECT c.id::text FROM carteiras c
WHERE c.rede_id = $1::uuid AND c.usuario_id = $2::uuid
FOR UPDATE`
	var carteiraID string
	err := q.QueryRowContext(ctx, lockCarteira, redeID, usuarioID).Scan(&carteiraID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSaldoInsuficiente
	}
	if err != nil {
		return err
	}

	if err := r.expirarLotesCarteira(ctx, q, redeID, carteiraID); err != nil {
		return err
	}

	const sumQ = `
SELECT COALESCE(SUM(t.valor_token * t.direcao), 0)::float8
FROM transacoes_carteira t
WHERE t.rede_id = $1::uuid AND t.carteira_id = $2::uuid`
	var saldo float64
	if err = q.QueryRowContext(ctx, sumQ, redeID, carteiraID).Scan(&saldo); err != nil {
		return err
	}
	if saldo < valorToken {
		return ErrSaldoInsuficiente
	}

	const ins = `
INSERT INTO transacoes_carteira (
  rede_id, carteira_id, tipo, valor_fiat, valor_token, direcao, tipo_referencia, referencia_id, metadados, ocorrido_em
) VALUES (
  $1::uuid, $2::uuid, 'AJUSTE'::tipo_transacao_carteira, 0, $3::numeric, -1, $4, $5::uuid, '{}'::jsonb, NOW()
) ON CONFLICT (rede_id, tipo_referencia, referencia_id, tipo) DO NOTHING`
	res, err := q.ExecContext(ctx, ins, redeID, carteiraID, valorToken, tipoReferencia, referenciaID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil
	}
	return r.consumirLotesFIFO(ctx, q, redeID, carteiraID, valorToken)
}

func (r *carteiraPostgres) DebitarMoeda(redeID, usuarioID string, valorToken float64, tipoReferencia, referenciaID string) error {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	tipoReferencia = strings.TrimSpace(tipoReferencia)
	referenciaID = strings.TrimSpace(referenciaID)
	if redeID == "" || usuarioID == "" || tipoReferencia == "" || referenciaID == "" {
		return errors.New("dados invalidos para debito")
	}
	if valorToken <= 0 {
		return errors.New("valor de debito deve ser positivo")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := r.debitarMoedaInterno(ctx, tx, redeID, usuarioID, valorToken, tipoReferencia, referenciaID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *carteiraPostgres) DebitarMoedaTx(ctx context.Context, tx *sql.Tx, redeID, usuarioID string, valorToken float64, tipoReferencia, referenciaID string) error {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	tipoReferencia = strings.TrimSpace(tipoReferencia)
	referenciaID = strings.TrimSpace(referenciaID)
	if redeID == "" || usuarioID == "" || tipoReferencia == "" || referenciaID == "" {
		return errors.New("dados invalidos para debito")
	}
	if valorToken <= 0 {
		return errors.New("valor de debito deve ser positivo")
	}
	return r.debitarMoedaInterno(ctx, tx, redeID, usuarioID, valorToken, tipoReferencia, referenciaID)
}

func (r *carteiraPostgres) expirarLotesCarteira(ctx context.Context, q dbExec, redeID, carteiraID string) error {
	rows, err := q.QueryContext(ctx, `
SELECT id::text, valor_restante::float8
FROM carteira_lotes
WHERE rede_id = $1::uuid AND carteira_id = $2::uuid
  AND valor_restante > 0 AND expira_em IS NOT NULL AND expira_em <= NOW()
FOR UPDATE`, redeID, carteiraID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type vencido struct {
		id  string
		val float64
	}
	var lista []vencido
	for rows.Next() {
		var v vencido
		if err := rows.Scan(&v.id, &v.val); err != nil {
			return err
		}
		lista = append(lista, v)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, v := range lista {
		if v.val <= 0 {
			continue
		}
		res, err := q.ExecContext(ctx, `
INSERT INTO transacoes_carteira (
  rede_id, carteira_id, tipo, valor_fiat, valor_token, direcao, tipo_referencia, referencia_id, metadados, ocorrido_em
) VALUES (
  $1::uuid, $2::uuid, 'EXPIRACAO'::tipo_transacao_carteira, 0, $3::numeric, -1, 'expiracao_lote', $4::uuid, '{}'::jsonb, NOW()
) ON CONFLICT (rede_id, tipo_referencia, referencia_id, tipo) DO NOTHING`, redeID, carteiraID, v.val, v.id)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			// Já expirado no razão — só garante lote zerado.
		}
		_, err = q.ExecContext(ctx, `
UPDATE carteira_lotes
SET valor_restante = 0, atualizado_em = NOW()
WHERE id = $1::uuid AND valor_restante > 0`, v.id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *carteiraPostgres) ExpirarLotesVencidos(redeID, usuarioID string) error {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return errors.New("ids invalidos")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var carteiraID string
	err = tx.QueryRowContext(ctx, `
SELECT id::text FROM carteiras
WHERE rede_id = $1::uuid AND usuario_id = $2::uuid
FOR UPDATE`, redeID, usuarioID).Scan(&carteiraID)
	if errors.Is(err, sql.ErrNoRows) {
		return tx.Commit()
	}
	if err != nil {
		return err
	}
	if err := r.expirarLotesCarteira(ctx, tx, redeID, carteiraID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *carteiraPostgres) ObterSaldoToken(redeID, usuarioID string) (float64, error) {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return 0, errors.New("ids invalidos")
	}
	_ = r.ExpirarLotesVencidos(redeID, usuarioID)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
SELECT COALESCE(SUM(t.valor_token * t.direcao), 0)::float8
FROM transacoes_carteira t
INNER JOIN carteiras c ON c.id = t.carteira_id AND c.rede_id = t.rede_id
WHERE t.rede_id = $1::uuid AND c.usuario_id = $2::uuid`
	var sal float64
	err := r.db.QueryRowContext(ctx, q, redeID, usuarioID).Scan(&sal)
	if err != nil {
		return 0, err
	}
	return sal, nil
}

func (r *carteiraPostgres) ListarLotesAtivos(redeID, usuarioID string) ([]CarteiraLoteAtivo, error) {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return nil, errors.New("ids invalidos")
	}
	_ = r.ExpirarLotesVencidos(redeID, usuarioID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, `
SELECT
  l.id::text,
  l.valor_restante::float8,
  l.valor_inicial::float8,
  l.origem_tipo,
  l.creditado_em,
  l.expira_em
FROM carteira_lotes l
INNER JOIN carteiras c ON c.id = l.carteira_id AND c.rede_id = l.rede_id
WHERE l.rede_id = $1::uuid AND c.usuario_id = $2::uuid AND l.valor_restante > 0
ORDER BY l.expira_em ASC NULLS LAST, l.creditado_em ASC, l.id ASC`, redeID, usuarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().UTC()
	out := make([]CarteiraLoteAtivo, 0)
	for rows.Next() {
		var item CarteiraLoteAtivo
		var expira sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.ValorRestante,
			&item.ValorInicial,
			&item.OrigemTipo,
			&item.CreditadoEm,
			&expira,
		); err != nil {
			return nil, err
		}
		if expira.Valid {
			t := expira.Time.UTC()
			item.ExpiraEm = &t
			sec := int64(t.Sub(now).Seconds())
			if sec < 0 {
				sec = 0
			}
			item.SegundosRestantes = &sec
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
