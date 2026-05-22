package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type eredeGatewayPostgres struct {
	db *sql.DB
}

func NovoERedeGatewayPostgres(db *sql.DB) ERedeGatewayRepositorio {
	return &eredeGatewayPostgres{db: db}
}

func (r *eredeGatewayPostgres) BuscarPorRedeID(idRede string) (*ERedeGatewayCredenciais, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, ErrERedeGatewayNaoConfigurado
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT COALESCE(TRIM(pv), ''), COALESCE(TRIM(client_secret), ''), COALESCE(NULLIF(TRIM(ambiente), ''), 'sandbox')
FROM rede_erede WHERE rede_id = $1`
	var c ERedeGatewayCredenciais
	err := r.db.QueryRowContext(ctx, q, idRede).Scan(&c.PV, &c.ClientSecret, &c.Ambiente)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrERedeGatewayNaoConfigurado
		}
		return nil, err
	}
	if c.PV == "" && c.ClientSecret == "" {
		return nil, ErrERedeGatewayNaoConfigurado
	}
	return &c, nil
}

func (r *eredeGatewayPostgres) Upsert(idRede, pv, clientSecret, ambiente string) error {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return errors.New("rede_id vazio")
	}
	ambiente = normalizarAmbienteERede(ambiente)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
INSERT INTO rede_erede (rede_id, pv, client_secret, ambiente, atualizado_em)
VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4, NOW())
ON CONFLICT (rede_id) DO UPDATE SET
  pv = NULLIF(EXCLUDED.pv, ''),
  client_secret = NULLIF(EXCLUDED.client_secret, ''),
  ambiente = EXCLUDED.ambiente,
  atualizado_em = NOW()`
	_, err := r.db.ExecContext(ctx, q, idRede, strings.TrimSpace(pv), strings.TrimSpace(clientSecret), ambiente)
	return err
}

func (r *eredeGatewayPostgres) BuscarPorPostoID(idPosto, idRede string) (*ERedeGatewayCredenciais, error) {
	idPosto = strings.TrimSpace(idPosto)
	idRede = strings.TrimSpace(idRede)
	if idPosto == "" || idRede == "" {
		return nil, ErrERedeGatewayPostoNaoConfigurado
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT COALESCE(TRIM(pe.pv), ''), COALESCE(TRIM(pe.client_secret), ''), COALESCE(NULLIF(TRIM(pe.ambiente), ''), 'sandbox')
FROM posto_erede pe
INNER JOIN postos p ON p.id = pe.posto_id AND p.rede_id = $2::uuid
WHERE pe.posto_id = $1::uuid`
	var c ERedeGatewayCredenciais
	err := r.db.QueryRowContext(ctx, q, idPosto, idRede).Scan(&c.PV, &c.ClientSecret, &c.Ambiente)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrERedeGatewayPostoNaoConfigurado
		}
		return nil, err
	}
	if c.PV == "" && c.ClientSecret == "" {
		return nil, ErrERedeGatewayPostoNaoConfigurado
	}
	return &c, nil
}

func (r *eredeGatewayPostgres) UpsertPosto(idPosto, idRede, pv, clientSecret, ambiente string) error {
	idPosto = strings.TrimSpace(idPosto)
	idRede = strings.TrimSpace(idRede)
	if idPosto == "" || idRede == "" {
		return errors.New("posto_id ou rede_id vazio")
	}
	ambiente = normalizarAmbienteERede(ambiente)
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
INSERT INTO posto_erede (posto_id, pv, client_secret, ambiente, atualizado_em)
VALUES ($1::uuid, NULLIF($2, ''), NULLIF($3, ''), $4, NOW())
ON CONFLICT (posto_id) DO UPDATE SET
  pv = NULLIF(EXCLUDED.pv, ''),
  client_secret = NULLIF(EXCLUDED.client_secret, ''),
  ambiente = EXCLUDED.ambiente,
  atualizado_em = NOW()`
	_, err := r.db.ExecContext(ctx, q, idPosto, strings.TrimSpace(pv), strings.TrimSpace(clientSecret), ambiente)
	return err
}

func (r *eredeGatewayPostgres) ListarStatusPostosPorRede(idRede string) ([]PostoERedeStatus, error) {
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
SELECT
  p.id::text, p.nome, p.codigo,
  (COALESCE(TRIM(pe.pv), '') <> '') AS pv_ok,
  (COALESCE(TRIM(pe.client_secret), '') <> '') AS secret_ok
FROM postos p
LEFT JOIN posto_erede pe ON pe.posto_id = p.id
WHERE p.rede_id = $1::uuid
ORDER BY p.nome ASC`
	rows, err := r.db.QueryContext(ctx, q, idRede)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PostoERedeStatus
	for rows.Next() {
		var s PostoERedeStatus
		if err := rows.Scan(&s.PostoID, &s.Nome, &s.Codigo, &s.PvConfigurado, &s.SecretConfigurado); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func normalizarAmbienteERede(a string) string {
	if strings.EqualFold(strings.TrimSpace(a), "producao") || strings.EqualFold(strings.TrimSpace(a), "production") {
		return "producao"
	}
	return "sandbox"
}
