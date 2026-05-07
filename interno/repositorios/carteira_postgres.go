package repositorios

// Modelo de razão (transacoes_carteira): o saldo é sempre SUM(valor_token * direcao).
// Créditos usam direcao +1 (ex.: BONUS de indique e ganhe); débitos usam direcao -1 (ex.: resgate de prémio).
// Qualquer nova funcionalidade que mova moeda virtual deve passar por INSERT aqui — nunca alterar saldo “direto” na tabela carteiras.
// Débitos com concorrência: DebitarMoeda bloqueia a linha da carteira (FOR UPDATE), recalcula o saldo na transação e só então insere.

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ErrSaldoInsuficiente quando o cliente não tem token suficiente para o débito pedido.
var ErrSaldoInsuficiente = errors.New("saldo em token insuficiente")

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
	// CreditarBonusTx igual a CreditarBonus mas na mesma transacao (ex.: check-in + registo).
	CreditarBonusTx(ctx context.Context, tx *sql.Tx, redeID, carteiraID string, valorToken float64, tipoRef, idRef string) error
	// CreditarCashbackTx registra cashback (valor_fiat + valor_token) na mesma transacao.
	CreditarCashbackTx(ctx context.Context, tx *sql.Tx, redeID, carteiraID string, valorFiat, valorToken float64, tipoRef, idRef string) error
	// ObterSaldoToken soma valor_token * direcao das transações da carteira do utilizador na rede.
	ObterSaldoToken(redeID, usuarioID string) (float64, error)
	// DebitarMoeda regista saída (direcao -1, tipo AJUSTE). Transação atómica: bloqueia carteira, verifica saldo, insere.
	// Idempotente: UNIQUE (rede_id, tipo_referencia, referencia_id, tipo) — repetir o mesmo pedido não duplica débito.
	// tipoReferencia: ex. "resgate_premio"; referenciaID: UUID do pedido de resgate (ou outro evento deduplicável).
	DebitarMoeda(redeID, usuarioID string, valorToken float64, tipoReferencia, referenciaID string) error
	// DebitarMoedaTx igual a DebitarMoeda, porém dentro de transação já aberta.
	DebitarMoedaTx(ctx context.Context, tx *sql.Tx, redeID, usuarioID string, valorToken float64, tipoReferencia, referenciaID string) error
}

type carteiraPostgres struct {
	db *sql.DB
}

func NovoCarteiraPostgres(db *sql.DB) CarteiraRepositorio {
	return &carteiraPostgres{db: db}
}

// ObterOuCriarCarteira cria [carteiras] com token da moeda virtual da rede.
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

// CreditarBonus grava BONUS; idempotente via UNIQUE (rede, tipo_referencia, ref, tipo).
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
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	const ins = `
INSERT INTO transacoes_carteira (
  rede_id, carteira_id, tipo, valor_fiat, valor_token, direcao, tipo_referencia, referencia_id, metadados, ocorrido_em
) VALUES (
  $1::uuid, $2::uuid, 'BONUS'::tipo_transacao_carteira, 0, $3::numeric, 1, $4, $5::uuid, '{}'::jsonb, NOW()
) ON CONFLICT (rede_id, tipo_referencia, referencia_id, tipo) DO NOTHING
`
	_, err := r.db.ExecContext(ctx, ins, redeID, carteiraID, valorToken, tipoRef, idRef)
	return err
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
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const ins = `
INSERT INTO transacoes_carteira (
  rede_id, carteira_id, tipo, valor_fiat, valor_token, direcao, tipo_referencia, referencia_id, metadados, ocorrido_em
) VALUES (
  $1::uuid, $2::uuid, 'CASHBACK'::tipo_transacao_carteira, $3::numeric, $4::numeric, 1, $5, $6::uuid, '{}'::jsonb, NOW()
) ON CONFLICT (rede_id, tipo_referencia, referencia_id, tipo) DO NOTHING
`
	_, err := r.db.ExecContext(ctx, ins, redeID, carteiraID, valorFiat, valorToken, tipoRef, idRef)
	return err
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
	const ins = `
INSERT INTO transacoes_carteira (
  rede_id, carteira_id, tipo, valor_fiat, valor_token, direcao, tipo_referencia, referencia_id, metadados, ocorrido_em
) VALUES (
  $1::uuid, $2::uuid, 'BONUS'::tipo_transacao_carteira, 0, $3::numeric, 1, $4, $5::uuid, '{}'::jsonb, NOW()
) ON CONFLICT (rede_id, tipo_referencia, referencia_id, tipo) DO NOTHING
`
	_, err := tx.ExecContext(ctx, ins, redeID, carteiraID, valorToken, tipoRef, idRef)
	return err
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
	const ins = `
INSERT INTO transacoes_carteira (
  rede_id, carteira_id, tipo, valor_fiat, valor_token, direcao, tipo_referencia, referencia_id, metadados, ocorrido_em
) VALUES (
  $1::uuid, $2::uuid, 'CASHBACK'::tipo_transacao_carteira, $3::numeric, $4::numeric, 1, $5, $6::uuid, '{}'::jsonb, NOW()
) ON CONFLICT (rede_id, tipo_referencia, referencia_id, tipo) DO NOTHING
`
	_, err := tx.ExecContext(ctx, ins, redeID, carteiraID, valorFiat, valorToken, tipoRef, idRef)
	return err
}

func (r *carteiraPostgres) ObterSaldoToken(redeID, usuarioID string) (float64, error) {
	redeID = strings.TrimSpace(redeID)
	usuarioID = strings.TrimSpace(usuarioID)
	if redeID == "" || usuarioID == "" {
		return 0, errors.New("ids invalidos")
	}
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

	const lockCarteira = `
SELECT c.id::text FROM carteiras c
WHERE c.rede_id = $1::uuid AND c.usuario_id = $2::uuid
FOR UPDATE`
	var carteiraID string
	err = tx.QueryRowContext(ctx, lockCarteira, redeID, usuarioID).Scan(&carteiraID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSaldoInsuficiente
	}
	if err != nil {
		return err
	}

	const sumQ = `
SELECT COALESCE(SUM(t.valor_token * t.direcao), 0)::float8
FROM transacoes_carteira t
WHERE t.rede_id = $1::uuid AND t.carteira_id = $2::uuid`
	var saldo float64
	if err = tx.QueryRowContext(ctx, sumQ, redeID, carteiraID).Scan(&saldo); err != nil {
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
	res, err := tx.ExecContext(ctx, ins, redeID, carteiraID, valorToken, tipoReferencia, referenciaID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	// 0 linhas: conflito de idempotência (mesmo resgate) — saldo já estava ok; trata-se como sucesso.
	if n == 0 {
		_ = tx.Commit()
		return nil
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
	const lockCarteira = `
SELECT c.id::text FROM carteiras c
WHERE c.rede_id = $1::uuid AND c.usuario_id = $2::uuid
FOR UPDATE`
	var carteiraID string
	err := tx.QueryRowContext(ctx, lockCarteira, redeID, usuarioID).Scan(&carteiraID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSaldoInsuficiente
	}
	if err != nil {
		return err
	}
	const sumQ = `
SELECT COALESCE(SUM(t.valor_token * t.direcao), 0)::float8
FROM transacoes_carteira t
WHERE t.rede_id = $1::uuid AND t.carteira_id = $2::uuid`
	var saldo float64
	if err = tx.QueryRowContext(ctx, sumQ, redeID, carteiraID).Scan(&saldo); err != nil {
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
	_, err = tx.ExecContext(ctx, ins, redeID, carteiraID, valorToken, tipoReferencia, referenciaID)
	return err
}
