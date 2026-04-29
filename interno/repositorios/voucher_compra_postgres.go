package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type voucherCompraPostgres struct {
	db *sql.DB
}

func NovoVoucherCompraPostgres(db *sql.DB) VoucherCompraRepositorio {
	return &voucherCompraPostgres{db: db}
}

func (r *voucherCompraPostgres) CriarPendenteComPix(x *VoucherCompraRegistro) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	camp := nullUUIDString(x.CampanhaID)
	var mpID any
	if x.MpPaymentID != nil {
		mpID = *x.MpPaymentID
	}
	ref := ""
	if x.ReferenciaPagamento != nil {
		ref = *x.ReferenciaPagamento
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO voucher_compras (
  id, rede_id, usuario_id, campanha_id, valor_solicitado, desconto_aplicado, valor_final, litros, status,
  mp_payment_id, referencia_pagamento, expira_pagamento_em, criado_em, atualizado_em
) VALUES (
  $1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9::status_voucher_compra,
  $10, $11, $12, NOW(), NOW()
)
RETURNING id::text, criado_em, atualizado_em
`, x.ID, x.RedeID, x.UsuarioID, camp, x.ValorSolicitado, x.DescontoAplicado, x.ValorFinal, nullFloat64Ptr(x.Litros), x.Status,
		mpID, nullStringPtr(ref), x.ExpiraPagamento,
	).Scan(&x.ID, &x.CriadoEm, &x.AtualizadoEm)
}

func nullStringPtr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullFloat64Ptr(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func nullUUIDString(p *string) any {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil
	}
	return strings.TrimSpace(*p)
}

type scannerVcr interface {
	Scan(dest ...any) error
}

func scanVcr(s scannerVcr, x *VoucherCompraRegistro) error {
	var camp, ref, cod sql.NullString
	var mpID sql.NullInt64
	var litros sql.NullFloat64
	var exPag, exRes sql.NullTime
	err := s.Scan(
		&x.ID, &x.RedeID, &x.UsuarioID, &camp, &x.ValorSolicitado, &x.DescontoAplicado, &x.ValorFinal, &litros, &x.Status,
		&mpID, &ref, &cod, &exPag, &exRes, &x.CriadoEm, &x.AtualizadoEm,
	)
	if err != nil {
		return err
	}
	if litros.Valid {
		v := litros.Float64
		x.Litros = &v
	}
	if camp.Valid && strings.TrimSpace(camp.String) != "" {
		v := camp.String
		x.CampanhaID = &v
	}
	if mpID.Valid {
		v := mpID.Int64
		x.MpPaymentID = &v
	}
	if ref.Valid {
		v := ref.String
		x.ReferenciaPagamento = &v
	}
	if cod.Valid {
		v := cod.String
		x.CodigoResgate = &v
	}
	if exPag.Valid {
		t := exPag.Time
		x.ExpiraPagamento = &t
	}
	if exRes.Valid {
		t := exRes.Time
		x.ExpiraResgate = &t
	}
	return nil
}

func scanVcrEquipe(s scannerVcr, x *VoucherCompraRegistro, clienteNome, clienteEmail *string, campanhaBaseDesconto, campanhaTitulo *string) error {
	var camp, ref, cod sql.NullString
	var mpID sql.NullInt64
	var litros sql.NullFloat64
	var exPag, exRes sql.NullTime
	var cbCamp, cbTit sql.NullString
	err := s.Scan(
		&x.ID, &x.RedeID, &x.UsuarioID, &camp, &x.ValorSolicitado, &x.DescontoAplicado, &x.ValorFinal, &litros, &x.Status,
		&mpID, &ref, &cod, &exPag, &exRes, &x.CriadoEm, &x.AtualizadoEm,
		clienteNome, clienteEmail,
		&cbCamp, &cbTit,
	)
	if err != nil {
		return err
	}
	if litros.Valid {
		v := litros.Float64
		x.Litros = &v
	}
	if camp.Valid && strings.TrimSpace(camp.String) != "" {
		v := camp.String
		x.CampanhaID = &v
	}
	if mpID.Valid {
		v := mpID.Int64
		x.MpPaymentID = &v
	}
	if ref.Valid {
		v := ref.String
		x.ReferenciaPagamento = &v
	}
	if cod.Valid {
		v := cod.String
		x.CodigoResgate = &v
	}
	if exPag.Valid {
		t := exPag.Time
		x.ExpiraPagamento = &t
	}
	if exRes.Valid {
		t := exRes.Time
		x.ExpiraResgate = &t
	}
	if campanhaBaseDesconto != nil {
		if cbCamp.Valid {
			*campanhaBaseDesconto = strings.TrimSpace(cbCamp.String)
		} else {
			*campanhaBaseDesconto = ""
		}
	}
	if campanhaTitulo != nil {
		if cbTit.Valid {
			*campanhaTitulo = strings.TrimSpace(cbTit.String)
		} else {
			*campanhaTitulo = ""
		}
	}
	return nil
}

func (r *voucherCompraPostgres) BuscarPorID(id, usuarioID, redeID string) (*VoucherCompraRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT
  id::text, rede_id::text, usuario_id::text, campanha_id::text,
  valor_solicitado, desconto_aplicado, valor_final, litros::float8, status::text,
  mp_payment_id, referencia_pagamento, codigo_resgate, expira_pagamento_em, expira_resgate_em, criado_em, atualizado_em
FROM voucher_compras
WHERE id = $1::uuid AND usuario_id = $2::uuid AND rede_id = $3::uuid`
	var x VoucherCompraRegistro
	err := scanVcr(r.db.QueryRowContext(ctx, q, id, usuarioID, redeID), &x)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVoucherCompraNaoEncontrado
		}
		return nil, err
	}
	return &x, nil
}

func (r *voucherCompraPostgres) ListarDoUsuario(redeID, usuarioID string, limite int) ([]*VoucherCompraRegistro, error) {
	if limite < 1 || limite > 200 {
		limite = 50
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, `
SELECT
  id::text, rede_id::text, usuario_id::text, campanha_id::text,
  valor_solicitado, desconto_aplicado, valor_final, litros::float8, status::text,
  mp_payment_id, referencia_pagamento, codigo_resgate, expira_pagamento_em, expira_resgate_em, criado_em, atualizado_em
FROM voucher_compras
WHERE rede_id = $1::uuid AND usuario_id = $2::uuid
ORDER BY criado_em DESC
LIMIT $3`, redeID, usuarioID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*VoucherCompraRegistro
	for rows.Next() {
		var x VoucherCompraRegistro
		if err := scanVcr(rows, &x); err != nil {
			return nil, err
		}
		out = append(out, &x)
	}
	return out, rows.Err()
}

func (r *voucherCompraPostgres) ContarUsosCampanhaUsuario(campanhaID, usuarioID, redeID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var n int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM voucher_compras
WHERE campanha_id = $1::uuid AND usuario_id = $2::uuid AND rede_id = $3::uuid
  AND status IN ('ATIVO', 'USADO')
`, campanhaID, usuarioID, redeID).Scan(&n)
	return n, err
}

func (r *voucherCompraPostgres) ListarUsosAprovadosPorCampanha(redeID, usuarioID string) (map[string]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, `
SELECT campanha_id::text, COUNT(*)::int
FROM voucher_compras
WHERE rede_id = $1::uuid AND usuario_id = $2::uuid
  AND campanha_id IS NOT NULL
  AND status IN ('ATIVO', 'USADO')
GROUP BY campanha_id
`, redeID, usuarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		if strings.TrimSpace(id) != "" {
			out[id] = n
		}
	}
	return out, rows.Err()
}

func (r *voucherCompraPostgres) BuscarPorIDRede(id, redeID string) (*VoucherCompraRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT
  id::text, rede_id::text, usuario_id::text, campanha_id::text,
  valor_solicitado, desconto_aplicado, valor_final, litros::float8, status::text,
  mp_payment_id, referencia_pagamento, codigo_resgate, expira_pagamento_em, expira_resgate_em, criado_em, atualizado_em
FROM voucher_compras
WHERE id = $1::uuid AND rede_id = $2::uuid`
	var x VoucherCompraRegistro
	err := scanVcr(r.db.QueryRowContext(ctx, q, id, redeID), &x)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVoucherCompraNaoEncontrado
		}
		return nil, err
	}
	return &x, nil
}

func (r *voucherCompraPostgres) BuscarPorCodigoResgateConsultaEquipe(codigo, redeID string) (*VoucherCompraConsultaEquipe, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	codigo = strings.TrimSpace(codigo)
	redeID = strings.TrimSpace(redeID)
	if codigo == "" || redeID == "" {
		return nil, ErrVoucherCompraNaoEncontrado
	}
	const q = `
SELECT
  v.id::text, v.rede_id::text, v.usuario_id::text, v.campanha_id::text,
  v.valor_solicitado, v.desconto_aplicado, v.valor_final, v.litros::float8, v.status::text,
  v.mp_payment_id, v.referencia_pagamento, v.codigo_resgate, v.expira_pagamento_em, v.expira_resgate_em, v.criado_em, v.atualizado_em,
  COALESCE(TRIM(u.nome_completo), ''),
  COALESCE(TRIM(u.email), ''),
  c.base_desconto,
  COALESCE(NULLIF(TRIM(c.titulo), ''), NULLIF(TRIM(c.nome), ''))
FROM voucher_compras v
INNER JOIN usuarios u ON u.id = v.usuario_id AND u.rede_id = v.rede_id
LEFT JOIN campanhas c ON c.id = v.campanha_id AND c.rede_id = v.rede_id
WHERE v.rede_id = $1::uuid
  AND v.codigo_resgate IS NOT NULL
  AND upper(trim(v.codigo_resgate)) = upper(trim($2))
LIMIT 1`
	var out VoucherCompraConsultaEquipe
	var nome, email string
	var baseCamp, titCamp string
	err := scanVcrEquipe(r.db.QueryRowContext(ctx, q, redeID, codigo), &out.VoucherCompraRegistro, &nome, &email, &baseCamp, &titCamp)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVoucherCompraNaoEncontrado
		}
		return nil, err
	}
	out.ClienteNomeCompleto = nome
	out.ClienteEmail = email
	out.CampanhaTitulo = titCamp
	out.TipoCompra = TipoCompraVoucher(out.Litros, baseCamp)
	return &out, nil
}

func (r *voucherCompraPostgres) AtivarPagamentoAprovado(id, redeID, codigo string, expiraResgate time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := r.db.ExecContext(ctx, `
UPDATE voucher_compras SET
  status = 'ATIVO',
  codigo_resgate = $3,
  expira_resgate_em = $4,
  atualizado_em = NOW()
WHERE id = $1::uuid AND rede_id = $2::uuid
  AND status = 'AGUARDANDO_PAGAMENTO'
`, id, redeID, strings.TrimSpace(codigo), expiraResgate)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("nenhuma linha ativada; status ou id invalido")
	}
	return nil
}

func scanVoucherPainelLinha(s scannerVcr, x *VoucherCompraPainelLinha) error {
	var camp, cod sql.NullString
	var litros sql.NullFloat64
	var exPag, exRes, usado sql.NullTime
	var postoNome sql.NullString
	var cbCamp, cbTit sql.NullString
	err := s.Scan(
		&x.ID, &x.UsuarioID, &camp,
		&x.ValorSolicitado, &x.DescontoAplicado, &x.ValorFinal, &litros, &x.Status,
		&cod, &exPag, &exRes, &usado, &x.CriadoEm, &x.AtualizadoEm,
		&x.ClienteNomeCompleto, &postoNome,
		&cbCamp, &cbTit,
	)
	if err != nil {
		return err
	}
	baseCamp := ""
	if cbCamp.Valid {
		baseCamp = strings.TrimSpace(cbCamp.String)
	}
	if cbTit.Valid {
		x.CampanhaTitulo = strings.TrimSpace(cbTit.String)
	}
	if litros.Valid {
		v := litros.Float64
		x.Litros = &v
	}
	x.TipoCompra = TipoCompraVoucher(x.Litros, baseCamp)
	if camp.Valid && strings.TrimSpace(camp.String) != "" {
		v := camp.String
		x.CampanhaID = &v
	}
	if cod.Valid && strings.TrimSpace(cod.String) != "" {
		v := cod.String
		x.CodigoResgate = &v
	}
	if exPag.Valid {
		t := exPag.Time
		x.ExpiraPagamento = &t
	}
	if exRes.Valid {
		t := exRes.Time
		x.ExpiraResgate = &t
	}
	if usado.Valid {
		t := usado.Time
		x.UsadoEm = &t
	}
	if postoNome.Valid {
		x.PostoUsoNome = strings.TrimSpace(postoNome.String)
	}
	return nil
}

func (r *voucherCompraPostgres) ListarPainelPorRede(redeID string, limite, offset int, statusFiltro string) ([]*VoucherCompraPainelLinha, int, error) {
	redeID = strings.TrimSpace(redeID)
	if redeID == "" {
		return nil, 0, errors.New("rede vazia")
	}
	if limite < 1 || limite > 200 {
		limite = 50
	}
	if offset < 0 {
		offset = 0
	}
	statusFiltro = strings.TrimSpace(statusFiltro)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var total int
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM voucher_compras v
WHERE v.rede_id = $1::uuid
  AND ($2 = '' OR v.status::text = $2)
`, redeID, statusFiltro).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT
  v.id::text, v.usuario_id::text, v.campanha_id::text,
  v.valor_solicitado, v.desconto_aplicado, v.valor_final, v.litros::float8, v.status::text,
  v.codigo_resgate, v.expira_pagamento_em, v.expira_resgate_em, v.usado_em, v.criado_em, v.atualizado_em,
  COALESCE(TRIM(u.nome_completo), ''),
  p.nome,
  c.base_desconto,
  COALESCE(NULLIF(TRIM(c.titulo), ''), NULLIF(TRIM(c.nome), ''))
FROM voucher_compras v
INNER JOIN usuarios u ON u.id = v.usuario_id AND u.rede_id = v.rede_id
LEFT JOIN postos p ON p.id = v.posto_id_uso AND p.rede_id = v.rede_id
LEFT JOIN campanhas c ON c.id = v.campanha_id AND c.rede_id = v.rede_id
WHERE v.rede_id = $1::uuid
  AND ($2 = '' OR v.status::text = $2)
ORDER BY v.criado_em DESC
LIMIT $3 OFFSET $4
`, redeID, statusFiltro, limite, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*VoucherCompraPainelLinha
	for rows.Next() {
		var x VoucherCompraPainelLinha
		if err := scanVoucherPainelLinha(rows, &x); err != nil {
			return nil, 0, err
		}
		out = append(out, &x)
	}
	return out, total, rows.Err()
}
