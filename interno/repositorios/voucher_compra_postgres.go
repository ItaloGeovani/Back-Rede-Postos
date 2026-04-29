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
	comb := nullUUIDString(x.CombustivelRedeID)
	return r.db.QueryRowContext(ctx, `
INSERT INTO voucher_compras (
  id, rede_id, usuario_id, campanha_id, combustivel_rede_id, valor_solicitado, desconto_aplicado, valor_final, litros, status,
  mp_payment_id, referencia_pagamento, expira_pagamento_em, criado_em, atualizado_em
) VALUES (
  $1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10::status_voucher_compra,
  $11, $12, $13, NOW(), NOW()
)
RETURNING id::text, criado_em, atualizado_em
`, x.ID, x.RedeID, x.UsuarioID, camp, comb, x.ValorSolicitado, x.DescontoAplicado, x.ValorFinal, nullFloat64Ptr(x.Litros), x.Status,
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
	var exPag, exRes, usado sql.NullTime
	var combID, combNome sql.NullString
	var postoID, postoNome, opUID, opPapel, opNome sql.NullString
	err := s.Scan(
		&x.ID, &x.RedeID, &x.UsuarioID, &camp, &x.ValorSolicitado, &x.DescontoAplicado, &x.ValorFinal, &litros, &x.Status,
		&mpID, &ref, &cod, &exPag, &exRes, &x.CriadoEm, &x.AtualizadoEm,
		&combID, &combNome,
		&usado, &postoID, &postoNome, &opUID, &opPapel, &opNome,
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
	preencherCombustivelRede(x, combID, combNome)
	preencherUsoPostoOperador(x, usado, postoID, postoNome, opUID, opPapel, opNome)
	return nil
}

func preencherTipoCampanhaDoJoin(x *VoucherCompraRegistro, cbCamp, cbTit sql.NullString) {
	baseCamp := ""
	if cbCamp.Valid {
		baseCamp = strings.TrimSpace(cbCamp.String)
	}
	x.CampanhaTitulo = ""
	if cbTit.Valid {
		x.CampanhaTitulo = strings.TrimSpace(cbTit.String)
	}
	x.TipoCompra = TipoCompraVoucher(x.Litros, baseCamp)
}

func preencherCombustivelRede(x *VoucherCompraRegistro, id, nome sql.NullString) {
	x.CombustivelRedeID = nil
	x.CombustivelRedeNome = ""
	if id.Valid && strings.TrimSpace(id.String) != "" {
		v := strings.TrimSpace(id.String)
		x.CombustivelRedeID = &v
	}
	if nome.Valid && strings.TrimSpace(nome.String) != "" {
		x.CombustivelRedeNome = strings.TrimSpace(nome.String)
	}
}

func preencherUsoPostoOperador(x *VoucherCompraRegistro, usado sql.NullTime, postoID, postoNome, opUID, opPapel, opNome sql.NullString) {
	x.UsadoEm = nil
	x.PostoUsoID = nil
	x.PostoUsoNome = ""
	x.OperadorUsuarioID = nil
	x.OperadorPapel = ""
	x.OperadorNomeSnapshot = ""
	if usado.Valid {
		t := usado.Time
		x.UsadoEm = &t
	}
	if postoID.Valid && strings.TrimSpace(postoID.String) != "" {
		v := strings.TrimSpace(postoID.String)
		x.PostoUsoID = &v
	}
	if postoNome.Valid {
		x.PostoUsoNome = strings.TrimSpace(postoNome.String)
	}
	if opUID.Valid && strings.TrimSpace(opUID.String) != "" {
		v := strings.TrimSpace(opUID.String)
		x.OperadorUsuarioID = &v
	}
	if opPapel.Valid {
		x.OperadorPapel = strings.TrimSpace(opPapel.String)
	}
	if opNome.Valid {
		x.OperadorNomeSnapshot = strings.TrimSpace(opNome.String)
	}
}

func scanVcrComCampanha(s scannerVcr, x *VoucherCompraRegistro) error {
	var camp, ref, cod sql.NullString
	var mpID sql.NullInt64
	var litros sql.NullFloat64
	var exPag, exRes, usado sql.NullTime
	var cbCamp, cbTit sql.NullString
	var combID, combNome sql.NullString
	var postoID, postoNome, opUID, opPapel, opNome sql.NullString
	err := s.Scan(
		&x.ID, &x.RedeID, &x.UsuarioID, &camp, &x.ValorSolicitado, &x.DescontoAplicado, &x.ValorFinal, &litros, &x.Status,
		&mpID, &ref, &cod, &exPag, &exRes, &x.CriadoEm, &x.AtualizadoEm,
		&cbCamp, &cbTit,
		&combID, &combNome,
		&usado, &postoID, &postoNome, &opUID, &opPapel, &opNome,
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
	preencherTipoCampanhaDoJoin(x, cbCamp, cbTit)
	preencherCombustivelRede(x, combID, combNome)
	preencherUsoPostoOperador(x, usado, postoID, postoNome, opUID, opPapel, opNome)
	return nil
}

func scanVcrEquipe(s scannerVcr, x *VoucherCompraRegistro, clienteNome, clienteEmail *string) error {
	var camp, ref, cod sql.NullString
	var mpID sql.NullInt64
	var litros sql.NullFloat64
	var exPag, exRes, usado sql.NullTime
	var cbCamp, cbTit sql.NullString
	var combID, combNome sql.NullString
	var postoID, postoNome, opUID, opPapel, opNome sql.NullString
	err := s.Scan(
		&x.ID, &x.RedeID, &x.UsuarioID, &camp, &x.ValorSolicitado, &x.DescontoAplicado, &x.ValorFinal, &litros, &x.Status,
		&mpID, &ref, &cod, &exPag, &exRes, &x.CriadoEm, &x.AtualizadoEm,
		clienteNome, clienteEmail,
		&cbCamp, &cbTit,
		&combID, &combNome,
		&usado, &postoID, &postoNome, &opUID, &opPapel, &opNome,
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
	preencherTipoCampanhaDoJoin(x, cbCamp, cbTit)
	preencherCombustivelRede(x, combID, combNome)
	preencherUsoPostoOperador(x, usado, postoID, postoNome, opUID, opPapel, opNome)
	return nil
}

func (r *voucherCompraPostgres) BuscarPorID(id, usuarioID, redeID string) (*VoucherCompraRegistro, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const q = `
SELECT
  v.id::text, v.rede_id::text, v.usuario_id::text, v.campanha_id::text,
  v.valor_solicitado, v.desconto_aplicado, v.valor_final, v.litros::float8, v.status::text,
  v.mp_payment_id, v.referencia_pagamento, v.codigo_resgate, v.expira_pagamento_em, v.expira_resgate_em, v.criado_em, v.atualizado_em,
  c.base_desconto,
  COALESCE(NULLIF(TRIM(c.titulo), ''), NULLIF(TRIM(c.nome), '')),
  v.combustivel_rede_id::text,
  COALESCE(NULLIF(TRIM(comb.nome), ''), ''),
  v.usado_em, v.posto_id_uso::text,
  COALESCE(NULLIF(TRIM(pu.nome), ''), ''),
  v.operador_usuario_id::text, v.operador_papel, v.operador_nome_snapshot
FROM voucher_compras v
LEFT JOIN campanhas c ON c.id = v.campanha_id AND c.rede_id = v.rede_id
LEFT JOIN rede_combustiveis comb ON comb.id = v.combustivel_rede_id AND comb.rede_id = v.rede_id
LEFT JOIN postos pu ON pu.id = v.posto_id_uso AND pu.rede_id = v.rede_id
WHERE v.id = $1::uuid AND v.usuario_id = $2::uuid AND v.rede_id = $3::uuid`
	var x VoucherCompraRegistro
	err := scanVcrComCampanha(r.db.QueryRowContext(ctx, q, id, usuarioID, redeID), &x)
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
  v.id::text, v.rede_id::text, v.usuario_id::text, v.campanha_id::text,
  v.valor_solicitado, v.desconto_aplicado, v.valor_final, v.litros::float8, v.status::text,
  v.mp_payment_id, v.referencia_pagamento, v.codigo_resgate, v.expira_pagamento_em, v.expira_resgate_em, v.criado_em, v.atualizado_em,
  c.base_desconto,
  COALESCE(NULLIF(TRIM(c.titulo), ''), NULLIF(TRIM(c.nome), '')),
  v.combustivel_rede_id::text,
  COALESCE(NULLIF(TRIM(comb.nome), ''), ''),
  v.usado_em, v.posto_id_uso::text,
  COALESCE(NULLIF(TRIM(pu.nome), ''), ''),
  v.operador_usuario_id::text, v.operador_papel, v.operador_nome_snapshot
FROM voucher_compras v
LEFT JOIN campanhas c ON c.id = v.campanha_id AND c.rede_id = v.rede_id
LEFT JOIN rede_combustiveis comb ON comb.id = v.combustivel_rede_id AND comb.rede_id = v.rede_id
LEFT JOIN postos pu ON pu.id = v.posto_id_uso AND pu.rede_id = v.rede_id
WHERE v.rede_id = $1::uuid AND v.usuario_id = $2::uuid
ORDER BY v.criado_em DESC
LIMIT $3`, redeID, usuarioID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*VoucherCompraRegistro
	for rows.Next() {
		var x VoucherCompraRegistro
		if err := scanVcrComCampanha(rows, &x); err != nil {
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
  v.id::text, v.rede_id::text, v.usuario_id::text, v.campanha_id::text,
  v.valor_solicitado, v.desconto_aplicado, v.valor_final, v.litros::float8, v.status::text,
  v.mp_payment_id, v.referencia_pagamento, v.codigo_resgate, v.expira_pagamento_em, v.expira_resgate_em, v.criado_em, v.atualizado_em,
  v.combustivel_rede_id::text,
  COALESCE(NULLIF(TRIM(comb.nome), ''), ''),
  v.usado_em, v.posto_id_uso::text,
  COALESCE(NULLIF(TRIM(pu.nome), ''), ''),
  v.operador_usuario_id::text, v.operador_papel, v.operador_nome_snapshot
FROM voucher_compras v
LEFT JOIN rede_combustiveis comb ON comb.id = v.combustivel_rede_id AND comb.rede_id = v.rede_id
LEFT JOIN postos pu ON pu.id = v.posto_id_uso AND pu.rede_id = v.rede_id
WHERE v.id = $1::uuid AND v.rede_id = $2::uuid`
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
  COALESCE(NULLIF(TRIM(c.titulo), ''), NULLIF(TRIM(c.nome), '')),
  v.combustivel_rede_id::text,
  COALESCE(NULLIF(TRIM(comb.nome), ''), ''),
  v.usado_em, v.posto_id_uso::text,
  COALESCE(NULLIF(TRIM(pu.nome), ''), ''),
  v.operador_usuario_id::text, v.operador_papel, v.operador_nome_snapshot
FROM voucher_compras v
INNER JOIN usuarios u ON u.id = v.usuario_id AND u.rede_id = v.rede_id
LEFT JOIN campanhas c ON c.id = v.campanha_id AND c.rede_id = v.rede_id
LEFT JOIN rede_combustiveis comb ON comb.id = v.combustivel_rede_id AND comb.rede_id = v.rede_id
LEFT JOIN postos pu ON pu.id = v.posto_id_uso AND pu.rede_id = v.rede_id
WHERE v.rede_id = $1::uuid
  AND v.codigo_resgate IS NOT NULL
  AND upper(trim(v.codigo_resgate)) = upper(trim($2))
LIMIT 1`
	var out VoucherCompraConsultaEquipe
	var nome, email string
	err := scanVcrEquipe(r.db.QueryRowContext(ctx, q, redeID, codigo), &out.VoucherCompraRegistro, &nome, &email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrVoucherCompraNaoEncontrado
		}
		return nil, err
	}
	out.ClienteNomeCompleto = nome
	out.ClienteEmail = email
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

func (r *voucherCompraPostgres) RegistrarBaixaUso(idVoucher, redeID string, idPosto *string, operadorUsuarioID, operadorPapel, operadorNome string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	idVoucher = strings.TrimSpace(idVoucher)
	redeID = strings.TrimSpace(redeID)
	operadorUsuarioID = strings.TrimSpace(operadorUsuarioID)
	if idVoucher == "" || redeID == "" || operadorUsuarioID == "" {
		return errors.New("dados invalidos para baixa de voucher")
	}
	var posto any
	if idPosto != nil && strings.TrimSpace(*idPosto) != "" {
		posto = strings.TrimSpace(*idPosto)
	} else {
		posto = nil
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE voucher_compras SET
  status = 'USADO',
  usado_em = NOW(),
  posto_id_uso = $3,
  operador_usuario_id = $4::uuid,
  operador_papel = COALESCE(
    (SELECT NULLIF(TRIM(u.papel::text), '') FROM usuarios u WHERE u.id = $4::uuid AND u.rede_id = $2::uuid LIMIT 1),
    NULLIF(TRIM($5), '')
  ),
  operador_nome_snapshot = COALESCE(
    (SELECT NULLIF(TRIM(u.nome_completo), '') FROM usuarios u WHERE u.id = $4::uuid AND u.rede_id = $2::uuid LIMIT 1),
    NULLIF(TRIM($6), '')
  ),
  atualizado_em = NOW()
WHERE id = $1::uuid AND rede_id = $2::uuid
  AND status = 'ATIVO'
  AND (expira_resgate_em IS NULL OR expira_resgate_em > NOW())
`, idVoucher, redeID, posto, operadorUsuarioID, strings.TrimSpace(operadorPapel), strings.TrimSpace(operadorNome))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrVoucherBaixaNaoPermitida
	}
	return nil
}

func scanVoucherPainelLinha(s scannerVcr, x *VoucherCompraPainelLinha) error {
	var camp, cod sql.NullString
	var litros sql.NullFloat64
	var exPag, exRes, usado sql.NullTime
	var postoNome sql.NullString
	var cbCamp, cbTit sql.NullString
	var combID, combNome sql.NullString
	var opUID, opPapel, opNome sql.NullString
	err := s.Scan(
		&x.ID, &x.UsuarioID, &camp,
		&x.ValorSolicitado, &x.DescontoAplicado, &x.ValorFinal, &litros, &x.Status,
		&cod, &exPag, &exRes, &usado, &x.CriadoEm, &x.AtualizadoEm,
		&x.ClienteNomeCompleto, &postoNome,
		&cbCamp, &cbTit,
		&combID, &combNome,
		&opUID, &opPapel, &opNome,
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
	x.CombustivelRedeID = nil
	x.CombustivelRedeNome = ""
	if combID.Valid && strings.TrimSpace(combID.String) != "" {
		v := strings.TrimSpace(combID.String)
		x.CombustivelRedeID = &v
	}
	if combNome.Valid {
		x.CombustivelRedeNome = strings.TrimSpace(combNome.String)
	}
	x.OperadorUsuarioID = nil
	x.OperadorPapel = ""
	x.OperadorNomeSnapshot = ""
	if opUID.Valid && strings.TrimSpace(opUID.String) != "" {
		v := strings.TrimSpace(opUID.String)
		x.OperadorUsuarioID = &v
	}
	if opPapel.Valid {
		x.OperadorPapel = strings.TrimSpace(opPapel.String)
	}
	if opNome.Valid {
		x.OperadorNomeSnapshot = strings.TrimSpace(opNome.String)
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
  COALESCE(NULLIF(TRIM(c.titulo), ''), NULLIF(TRIM(c.nome), '')),
  v.combustivel_rede_id::text,
  COALESCE(NULLIF(TRIM(comb.nome), ''), ''),
  v.operador_usuario_id::text, v.operador_papel, v.operador_nome_snapshot
FROM voucher_compras v
INNER JOIN usuarios u ON u.id = v.usuario_id AND u.rede_id = v.rede_id
LEFT JOIN postos p ON p.id = v.posto_id_uso AND p.rede_id = v.rede_id
LEFT JOIN campanhas c ON c.id = v.campanha_id AND c.rede_id = v.rede_id
LEFT JOIN rede_combustiveis comb ON comb.id = v.combustivel_rede_id AND comb.rede_id = v.rede_id
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
