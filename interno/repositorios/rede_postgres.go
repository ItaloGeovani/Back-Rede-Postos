package repositorios

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"

	"github.com/jackc/pgx/v5/pgconn"
)

type redePostgres struct {
	db *sql.DB
}

func NovoRedePostgres(db *sql.DB) RedeRepositorio {
	return &redePostgres{db: db}
}

func (r *redePostgres) Listar() ([]*modelos.Rede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  id::text,
  nome_fantasia,
  razao_social,
  COALESCE(cnpj, ''),
  COALESCE(email_contato, ''),
  COALESCE(telefone, ''),
  COALESCE(valor_implantacao, 0),
  COALESCE(valor_mensalidade, 0),
  primeiro_cobranca,
  COALESCE(dia_cobranca, 0),
  moeda_virtual_nome,
  moeda_virtual_cotacao::float8,
  COALESCE(moeda_virtual_expira_dias, 0),
  COALESCE(voucher_dias_validade_resgate, 7),
  COALESCE(voucher_minutos_expira_pagamento_pix, 30),
  COALESCE(gateway_pagamento_modo, 'REDE'),
  COALESCE(gateway_provedor_ativo, 'MERCADO_PAGO'),
  COALESCE(gateway_meios_habilitados, '{"pix":true}'::jsonb),
  COALESCE(app_modulo_indique_ganhe, false),
  COALESCE(app_modulo_checkin_diario, false),
  COALESCE(app_modulo_gire_ganhe, false),
  COALESCE(app_modulo_redes_sociais, false),
  COALESCE(ativa, true),
  criado_em,
  atualizado_em
FROM redes
ORDER BY criado_em ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []*modelos.Rede
	for rows.Next() {
		item, err := scanRede(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return lista, nil
}

func (r *redePostgres) BuscarPorID(id string) (*modelos.Rede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
SELECT
  id::text,
  nome_fantasia,
  razao_social,
  COALESCE(cnpj, ''),
  COALESCE(email_contato, ''),
  COALESCE(telefone, ''),
  COALESCE(valor_implantacao, 0),
  COALESCE(valor_mensalidade, 0),
  primeiro_cobranca,
  COALESCE(dia_cobranca, 0),
  moeda_virtual_nome,
  moeda_virtual_cotacao::float8,
  COALESCE(moeda_virtual_expira_dias, 0),
  COALESCE(voucher_dias_validade_resgate, 7),
  COALESCE(voucher_minutos_expira_pagamento_pix, 30),
  COALESCE(gateway_pagamento_modo, 'REDE'),
  COALESCE(gateway_provedor_ativo, 'MERCADO_PAGO'),
  COALESCE(gateway_meios_habilitados, '{"pix":true}'::jsonb),
  COALESCE(app_modulo_indique_ganhe, false),
  COALESCE(app_modulo_checkin_diario, false),
  COALESCE(app_modulo_gire_ganhe, false),
  COALESCE(app_modulo_redes_sociais, false),
  COALESCE(ativa, true),
  criado_em,
  atualizado_em
FROM redes
WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	rede, err := scanRede(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRedeNaoEncontrada
		}
		return nil, err
	}
	return rede, nil
}

func (r *redePostgres) Criar(rede *modelos.Rede) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
INSERT INTO redes (
  nome_fantasia,
  razao_social,
  cnpj,
  email_contato,
  telefone,
  ativa,
  status,
  valor_implantacao,
  valor_mensalidade,
  primeiro_cobranca,
  dia_cobranca
)
VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, $9, $10, $11)
RETURNING id::text, criado_em, atualizado_em`

	status := "INATIVA"
	if rede.Ativa {
		status = "ATIVA"
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(rede.NomeFantasia),
		strings.TrimSpace(rede.RazaoSocial),
		strings.TrimSpace(rede.CNPJ),
		strings.TrimSpace(rede.EmailContato),
		strings.TrimSpace(rede.Telefone),
		rede.Ativa,
		status,
		rede.ValorImplantacao,
		rede.ValorMensalidade,
		rede.PrimeiroCobranca,
		rede.DiaCobranca,
	).Scan(&rede.ID, &rede.CriadoEm, &rede.AtualizadoEm)
	if err != nil {
		return mapearErroRedePostgres(err)
	}
	return nil
}

func (r *redePostgres) Atualizar(id string, atualizar func(*modelos.Rede) error) (*modelos.Rede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const queryBusca = `
SELECT
  id::text,
  nome_fantasia,
  razao_social,
  COALESCE(cnpj, ''),
  COALESCE(email_contato, ''),
  COALESCE(telefone, ''),
  COALESCE(valor_implantacao, 0),
  COALESCE(valor_mensalidade, 0),
  primeiro_cobranca,
  COALESCE(dia_cobranca, 0),
  moeda_virtual_nome,
  moeda_virtual_cotacao::float8,
  COALESCE(moeda_virtual_expira_dias, 0),
  COALESCE(voucher_dias_validade_resgate, 7),
  COALESCE(voucher_minutos_expira_pagamento_pix, 30),
  COALESCE(gateway_pagamento_modo, 'REDE'),
  COALESCE(gateway_provedor_ativo, 'MERCADO_PAGO'),
  COALESCE(gateway_meios_habilitados, '{"pix":true}'::jsonb),
  COALESCE(app_modulo_indique_ganhe, false),
  COALESCE(app_modulo_checkin_diario, false),
  COALESCE(app_modulo_gire_ganhe, false),
  COALESCE(app_modulo_redes_sociais, false),
  COALESCE(ativa, true),
  criado_em,
  atualizado_em
FROM redes
WHERE id = $1
FOR UPDATE`

	rede, err := scanRede(tx.QueryRowContext(ctx, queryBusca, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRedeNaoEncontrada
		}
		return nil, err
	}

	if err := atualizar(rede); err != nil {
		return nil, err
	}

	const queryAtualiza = `
UPDATE redes
SET
  nome_fantasia = $2,
  razao_social = $3,
  cnpj = NULLIF($4, ''),
  email_contato = NULLIF($5, ''),
  telefone = NULLIF($6, ''),
  ativa = $7,
  status = $8,
  valor_implantacao = $9,
  valor_mensalidade = $10,
  primeiro_cobranca = $11,
  dia_cobranca = $12,
  moeda_virtual_nome = $13,
  moeda_virtual_cotacao = $14,
  moeda_virtual_expira_dias = NULLIF($15, 0),
  voucher_dias_validade_resgate = $16,
  voucher_minutos_expira_pagamento_pix = $17,
  gateway_pagamento_modo = $18,
  gateway_provedor_ativo = $19,
  gateway_meios_habilitados = $20::jsonb,
  app_modulo_indique_ganhe = $21,
  app_modulo_checkin_diario = $22,
  app_modulo_gire_ganhe = $23,
  app_modulo_redes_sociais = $24,
  atualizado_em = NOW()
WHERE id = $1
RETURNING atualizado_em`

	status := "INATIVA"
	if rede.Ativa {
		status = "ATIVA"
	}

	err = tx.QueryRowContext(
		ctx,
		queryAtualiza,
		id,
		strings.TrimSpace(rede.NomeFantasia),
		strings.TrimSpace(rede.RazaoSocial),
		strings.TrimSpace(rede.CNPJ),
		strings.TrimSpace(rede.EmailContato),
		strings.TrimSpace(rede.Telefone),
		rede.Ativa,
		status,
		rede.ValorImplantacao,
		rede.ValorMensalidade,
		rede.PrimeiroCobranca,
		rede.DiaCobranca,
		strings.TrimSpace(rede.MoedaVirtualNome),
		rede.MoedaVirtualCotacao,
		rede.MoedaVirtualExpiraDias,
		rede.VoucherDiasValidadeResgate,
		rede.VoucherMinutosExpiraPagamentoPix,
		normalizarGatewayPagamentoModoRede(rede.GatewayPagamentoModo),
		normalizarGatewayProvedorRede(rede.GatewayProvedorAtivo),
		mustGatewayMeiosJSON(rede.GatewayMeiosHabilitados),
		rede.AppModuloIndiqueGanhe,
		rede.AppModuloCheckinDiario,
		rede.AppModuloGireGanhe,
		rede.AppModuloRedesSociais,
	).Scan(&rede.AtualizadoEm)
	if err != nil {
		return nil, mapearErroRedePostgres(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return rede, nil
}

type scannerRede interface {
	Scan(dest ...any) error
}

func scanRede(s scannerRede) (*modelos.Rede, error) {
	var rede modelos.Rede
	var primeiro sql.NullTime
	var meiosRaw []byte

	err := s.Scan(
		&rede.ID,
		&rede.NomeFantasia,
		&rede.RazaoSocial,
		&rede.CNPJ,
		&rede.EmailContato,
		&rede.Telefone,
		&rede.ValorImplantacao,
		&rede.ValorMensalidade,
		&primeiro,
		&rede.DiaCobranca,
		&rede.MoedaVirtualNome,
		&rede.MoedaVirtualCotacao,
		&rede.MoedaVirtualExpiraDias,
		&rede.VoucherDiasValidadeResgate,
		&rede.VoucherMinutosExpiraPagamentoPix,
		&rede.GatewayPagamentoModo,
		&rede.GatewayProvedorAtivo,
		&meiosRaw,
		&rede.AppModuloIndiqueGanhe,
		&rede.AppModuloCheckinDiario,
		&rede.AppModuloGireGanhe,
		&rede.AppModuloRedesSociais,
		&rede.Ativa,
		&rede.CriadoEm,
		&rede.AtualizadoEm,
	)
	if err != nil {
		return nil, err
	}
	if primeiro.Valid {
		rede.PrimeiroCobranca = primeiro.Time
	}
	if strings.TrimSpace(rede.GatewayPagamentoModo) == "" {
		rede.GatewayPagamentoModo = modelos.GatewayPagamentoModoRede
	}
	rede.GatewayProvedorAtivo = normalizarGatewayProvedorRede(rede.GatewayProvedorAtivo)
	rede.GatewayMeiosHabilitados = modelos.ParseGatewayMeiosJSON(meiosRaw)
	return &rede, nil
}

func normalizarGatewayProvedorRede(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == modelos.GatewayProvedorERede {
		return modelos.GatewayProvedorERede
	}
	return modelos.GatewayProvedorMercadoPago
}

func mustGatewayMeiosJSON(m modelos.GatewayMeiosHabilitados) []byte {
	b, err := json.Marshal(m)
	if err != nil || len(b) == 0 {
		b, _ = json.Marshal(modelos.MeiosPadrao())
	}
	return b
}

func mapearErroRedePostgres(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code != "23505" {
		return err
	}

	switch pgErr.ConstraintName {
	case "uq_redes_cnpj":
		return ErrRedeCNPJDuplicado
	case "uq_redes_nome_fantasia_lower":
		return ErrRedeNomeDuplicado
	default:
		msg := strings.ToLower(pgErr.Message)
		if strings.Contains(msg, "cnpj") {
			return ErrRedeCNPJDuplicado
		}
		if strings.Contains(msg, "nome") {
			return ErrRedeNomeDuplicado
		}
		return err
	}
}

func normalizarGatewayPagamentoModoRede(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == modelos.GatewayPagamentoModoPosto {
		return modelos.GatewayPagamentoModoPosto
	}
	return modelos.GatewayPagamentoModoRede
}
