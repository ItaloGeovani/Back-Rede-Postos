package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type mercadoPagoGatewayPostgres struct {
	db *sql.DB
}

func NovoMercadoPagoGatewayPostgres(db *sql.DB) MercadoPagoGatewayRepositorio {
	return &mercadoPagoGatewayPostgres{db: db}
}

func (r *mercadoPagoGatewayPostgres) BuscarPorRedeID(idRede string) (*MercadoPagoGatewayCredenciais, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrMercadoPagoGatewayNaoConfigurado
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const q = `
SELECT
  COALESCE(TRIM(mp_access_token), ''),
  COALESCE(TRIM(mp_webhook_secret), '')
FROM rede_mercado_pago
WHERE rede_id = $1`

	var c MercadoPagoGatewayCredenciais
	err := r.db.QueryRowContext(ctx, q, idRede).Scan(&c.AccessToken, &c.WebhookSecret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMercadoPagoGatewayNaoConfigurado
		}
		return nil, err
	}
	if c.AccessToken == "" && c.WebhookSecret == "" {
		return nil, ErrMercadoPagoGatewayNaoConfigurado
	}
	return &c, nil
}

func (r *mercadoPagoGatewayPostgres) Upsert(idRede, accessToken, webhookSecret string) error {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return errors.New("rede_id vazio")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	at := strings.TrimSpace(accessToken)
	ws := strings.TrimSpace(webhookSecret)

	const q = `
INSERT INTO rede_mercado_pago (rede_id, mp_access_token, mp_webhook_secret, atualizado_em)
VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NOW())
ON CONFLICT (rede_id) DO UPDATE SET
  mp_access_token = NULLIF(EXCLUDED.mp_access_token, ''),
  mp_webhook_secret = NULLIF(EXCLUDED.mp_webhook_secret, ''),
  atualizado_em = NOW()`

	_, err := r.db.ExecContext(ctx, q, idRede, at, ws)
	return err
}

func (r *mercadoPagoGatewayPostgres) BuscarPorPostoID(idPosto, idRede string) (*MercadoPagoGatewayCredenciais, error) {
	idPosto = strings.TrimSpace(idPosto)
	idRede = strings.TrimSpace(idRede)
	if idPosto == "" || idRede == "" {
		return nil, ErrMercadoPagoGatewayPostoNaoConfigurado
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const q = `
SELECT
  COALESCE(TRIM(pmp.mp_access_token), ''),
  COALESCE(TRIM(pmp.mp_webhook_secret), '')
FROM posto_mercado_pago pmp
INNER JOIN postos p ON p.id = pmp.posto_id AND p.rede_id = $2::uuid
WHERE pmp.posto_id = $1::uuid`

	var c MercadoPagoGatewayCredenciais
	err := r.db.QueryRowContext(ctx, q, idPosto, idRede).Scan(&c.AccessToken, &c.WebhookSecret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMercadoPagoGatewayPostoNaoConfigurado
		}
		return nil, err
	}
	if c.AccessToken == "" && c.WebhookSecret == "" {
		return nil, ErrMercadoPagoGatewayPostoNaoConfigurado
	}
	return &c, nil
}

func (r *mercadoPagoGatewayPostgres) UpsertPosto(idPosto, idRede, accessToken, webhookSecret string) error {
	idPosto = strings.TrimSpace(idPosto)
	idRede = strings.TrimSpace(idRede)
	if idPosto == "" || idRede == "" {
		return errors.New("posto_id ou rede_id vazio")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const existe = `SELECT 1 FROM postos WHERE id = $1::uuid AND rede_id = $2::uuid`
	var one int
	if err := r.db.QueryRowContext(ctx, existe, idPosto, idRede).Scan(&one); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("posto nao pertence a rede")
		}
		return err
	}

	const q = `
INSERT INTO posto_mercado_pago (posto_id, mp_access_token, mp_webhook_secret, atualizado_em)
VALUES ($1::uuid, NULLIF($2, ''), NULLIF($3, ''), NOW())
ON CONFLICT (posto_id) DO UPDATE SET
  mp_access_token = NULLIF(EXCLUDED.mp_access_token, ''),
  mp_webhook_secret = NULLIF(EXCLUDED.mp_webhook_secret, ''),
  atualizado_em = NOW()`
	_, err := r.db.ExecContext(ctx, q, idPosto, strings.TrimSpace(accessToken), strings.TrimSpace(webhookSecret))
	return err
}

func (r *mercadoPagoGatewayPostgres) ListarStatusPostosPorRede(idRede string) ([]PostoMercadoPagoStatus, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	const q = `
SELECT
  p.id::text,
  p.nome,
  p.codigo,
  (COALESCE(TRIM(pmp.mp_access_token), '') <> '') AS token_ok,
  (COALESCE(TRIM(pmp.mp_webhook_secret), '') <> '') AS secret_ok
FROM postos p
LEFT JOIN posto_mercado_pago pmp ON pmp.posto_id = p.id
WHERE p.rede_id = $1::uuid
ORDER BY p.nome ASC`

	rows, err := r.db.QueryContext(ctx, q, idRede)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PostoMercadoPagoStatus
	for rows.Next() {
		var s PostoMercadoPagoStatus
		if err := rows.Scan(&s.PostoID, &s.Nome, &s.Codigo, &s.MpAccessTokenOK, &s.MpWebhookSecretOK); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
