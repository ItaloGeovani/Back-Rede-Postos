package repositorios

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

var ErrWhatsAppConfigNaoEncontrada = errors.New("config whatsapp nao encontrada")

type EventosOperacionaisRepositorio interface {
	Inserir(e *modelos.EventoOperacional) error
	ListarPorRede(idRede string, limite, offset int) ([]*modelos.EventoOperacional, int, error)
}

type WhatsAppNotificacoesRepositorio interface {
	BuscarPorRede(idRede string) (*modelos.RedeWhatsAppNotificacoes, error)
	Upsert(c *modelos.RedeWhatsAppNotificacoes) error
}

type eventosOperacionaisPostgres struct {
	db *sql.DB
}

func NovoEventosOperacionaisPostgres(db *sql.DB) EventosOperacionaisRepositorio {
	return &eventosOperacionaisPostgres{db: db}
}

func (r *eventosOperacionaisPostgres) Inserir(e *modelos.EventoOperacional) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	payload := e.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	var posto any
	if e.IDPosto != nil && strings.TrimSpace(*e.IDPosto) != "" {
		posto = strings.TrimSpace(*e.IDPosto)
	}
	var entID any
	if e.IDEntidade != nil && strings.TrimSpace(*e.IDEntidade) != "" {
		entID = strings.TrimSpace(*e.IDEntidade)
	}
	const q = `
INSERT INTO rede_eventos_operacionais (
  rede_id, posto_id, tipo_evento, entidade_tipo, entidade_id, titulo, payload
) VALUES (
  $1::uuid, $2::uuid, $3, $4, $5::uuid, $6, $7::jsonb
)
RETURNING id::text, criado_em`
	return r.db.QueryRowContext(
		ctx, q,
		strings.TrimSpace(e.IDRede), posto, strings.TrimSpace(e.TipoEvento),
		strings.TrimSpace(e.EntidadeTipo), entID, strings.TrimSpace(e.Titulo), payload,
	).Scan(&e.ID, &e.CriadoEm)
}

func (r *eventosOperacionaisPostgres) ListarPorRede(idRede string, limite, offset int) ([]*modelos.EventoOperacional, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if limite < 1 {
		limite = 50
	}
	if limite > 200 {
		limite = 200
	}
	if offset < 0 {
		offset = 0
	}
	idRede = strings.TrimSpace(idRede)
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rede_eventos_operacionais WHERE rede_id = $1::uuid`, idRede).Scan(&total); err != nil {
		return nil, 0, err
	}
	const q = `
SELECT
  e.id::text,
  e.rede_id::text,
  e.posto_id::text,
  COALESCE(NULLIF(TRIM(p.nome_fantasia), ''), NULLIF(TRIM(p.nome), ''), ''),
  e.tipo_evento,
  e.entidade_tipo,
  e.entidade_id::text,
  e.titulo,
  e.payload,
  e.criado_em
FROM rede_eventos_operacionais e
LEFT JOIN postos p ON p.id = e.posto_id
WHERE e.rede_id = $1::uuid
ORDER BY e.criado_em DESC
LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, q, idRede, limite, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var lista []*modelos.EventoOperacional
	for rows.Next() {
		var e modelos.EventoOperacional
		var postoID, entID sql.NullString
		var payload []byte
		if err := rows.Scan(
			&e.ID, &e.IDRede, &postoID, &e.PostoNome, &e.TipoEvento, &e.EntidadeTipo,
			&entID, &e.Titulo, &payload, &e.CriadoEm,
		); err != nil {
			return nil, 0, err
		}
		if postoID.Valid && postoID.String != "" {
			s := postoID.String
			e.IDPosto = &s
		}
		if entID.Valid && entID.String != "" {
			s := entID.String
			e.IDEntidade = &s
		}
		e.Payload = json.RawMessage(payload)
		e.DadosNovos = e.Payload
		lista = append(lista, &e)
	}
	return lista, total, rows.Err()
}

type whatsAppNotificacoesPostgres struct {
	db *sql.DB
}

func NovoWhatsAppNotificacoesPostgres(db *sql.DB) WhatsAppNotificacoesRepositorio {
	return &whatsAppNotificacoesPostgres{db: db}
}

func (r *whatsAppNotificacoesPostgres) BuscarPorRede(idRede string) (*modelos.RedeWhatsAppNotificacoes, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	const q = `
SELECT
  rede_id::text,
  habilitado,
  COALESCE(instance_name, ''),
  COALESCE(instance_token, ''),
  COALESCE(group_jid, ''),
  notify_voucher_gerado,
  notify_voucher_pago,
  notify_voucher_baixa,
  notify_campanha,
  atualizado_em
FROM rede_whatsapp_notificacoes
WHERE rede_id = $1::uuid`
	var c modelos.RedeWhatsAppNotificacoes
	err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(idRede)).Scan(
		&c.IDRede, &c.Habilitado, &c.InstanceName, &c.InstanceToken, &c.GroupJID,
		&c.NotifyVoucherGerado, &c.NotifyVoucherPago, &c.NotifyVoucherBaixa, &c.NotifyCampanha,
		&c.AtualizadoEm,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return &modelos.RedeWhatsAppNotificacoes{
			IDRede:              strings.TrimSpace(idRede),
			NotifyVoucherGerado: true,
			NotifyVoucherPago:   true,
			NotifyVoucherBaixa:  true,
			NotifyCampanha:      true,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *whatsAppNotificacoesPostgres) Upsert(c *modelos.RedeWhatsAppNotificacoes) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
INSERT INTO rede_whatsapp_notificacoes (
  rede_id, habilitado, instance_name, instance_token, group_jid,
  notify_voucher_gerado, notify_voucher_pago, notify_voucher_baixa, notify_campanha, atualizado_em
) VALUES (
  $1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, NOW()
)
ON CONFLICT (rede_id) DO UPDATE SET
  habilitado = EXCLUDED.habilitado,
  instance_name = EXCLUDED.instance_name,
  instance_token = CASE
    WHEN EXCLUDED.instance_token = '' THEN rede_whatsapp_notificacoes.instance_token
    ELSE EXCLUDED.instance_token
  END,
  group_jid = EXCLUDED.group_jid,
  notify_voucher_gerado = EXCLUDED.notify_voucher_gerado,
  notify_voucher_pago = EXCLUDED.notify_voucher_pago,
  notify_voucher_baixa = EXCLUDED.notify_voucher_baixa,
  notify_campanha = EXCLUDED.notify_campanha,
  atualizado_em = NOW()
RETURNING atualizado_em`
	return r.db.QueryRowContext(
		ctx, q,
		strings.TrimSpace(c.IDRede), c.Habilitado,
		strings.TrimSpace(c.InstanceName), strings.TrimSpace(c.InstanceToken), strings.TrimSpace(c.GroupJID),
		c.NotifyVoucherGerado, c.NotifyVoucherPago, c.NotifyVoucherBaixa, c.NotifyCampanha,
	).Scan(&c.AtualizadoEm)
}
