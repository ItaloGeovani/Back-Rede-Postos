package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
)

var (
	ErrPremioResgateNaoEncontrado = errors.New("resgate de premio nao encontrado")
	ErrPremioResgateStatus        = errors.New("status do resgate nao permite a operacao")
)

type PremioResgateRepositorio interface {
	CriarTx(ctx context.Context, tx *sql.Tx, r *modelos.PremioResgate) error
	BuscarPorIDNaRede(id, idRede string) (*modelos.PremioResgate, error)
	ListarPorUsuario(idRede, usuarioID string) ([]*modelos.PremioResgate, error)
	ListarPorRede(idRede, status string, limite, offset int) ([]*modelos.PremioResgate, int, error)
	MarcarEntregue(id, idRede string, postoID *string, operadorUsuarioID, operadorPapel, operadorNome string) error
	MarcarCancelado(id, idRede, motivo string) error
}

type premioResgatePostgres struct {
	db *sql.DB
}

func NovoPremioResgatePostgres(db *sql.DB) PremioResgateRepositorio {
	return &premioResgatePostgres{db: db}
}

func scanPremioResgate(sc interface {
	Scan(dest ...any) error
}) (*modelos.PremioResgate, error) {
	var r modelos.PremioResgate
	var img sql.NullString
	var entregue, cancelado sql.NullTime
	var postoID, opUID sql.NullString
	var opPapel, opNome, motivo, postoNome, cliNome, cliEmail sql.NullString
	err := sc.Scan(
		&r.ID, &r.IDRede, &r.PremioID, &r.UsuarioID,
		&r.TituloSnapshot, &img, &r.ValorMoeda, &r.Status, &r.PrazoRetiradaEm, &r.CriadoEm,
		&entregue, &cancelado, &postoID, &opUID, &opPapel, &opNome, &motivo,
		&postoNome, &cliNome, &cliEmail,
	)
	if err != nil {
		return nil, err
	}
	if img.Valid {
		r.ImagemURLSnapshot = img.String
	}
	if entregue.Valid {
		t := entregue.Time
		r.EntregueEm = &t
	}
	if cancelado.Valid {
		t := cancelado.Time
		r.CanceladoEm = &t
	}
	if postoID.Valid {
		s := postoID.String
		r.PostoEntregaID = &s
	}
	if opUID.Valid {
		s := opUID.String
		r.OperadorUsuarioID = &s
	}
	r.OperadorPapel = opPapel.String
	r.OperadorNomeSnapshot = opNome.String
	r.MotivoCancelamento = motivo.String
	r.PostoEntregaNome = postoNome.String
	r.ClienteNomeCompleto = cliNome.String
	r.ClienteEmail = cliEmail.String
	r.PrazoVencido = r.Status == modelos.PremioResgateAguardandoRetirada && time.Now().After(r.PrazoRetiradaEm)
	return &r, nil
}

const premioResgateSelect = `
SELECT
  r.id::text,
  r.rede_id::text,
  r.premio_id::text,
  r.usuario_id::text,
  r.titulo_snapshot,
  r.imagem_url_snapshot,
  r.valor_moeda::float8,
  r.status,
  r.prazo_retirada_em,
  r.criado_em,
  r.entregue_em,
  r.cancelado_em,
  r.posto_entrega_id::text,
  r.operador_usuario_id::text,
  COALESCE(r.operador_papel, ''),
  COALESCE(r.operador_nome_snapshot, ''),
  COALESCE(r.motivo_cancelamento, ''),
  COALESCE(NULLIF(TRIM(p.nome_fantasia), ''), NULLIF(TRIM(p.nome), ''), ''),
  COALESCE(u.nome_completo, ''),
  COALESCE(u.email, '')
FROM premio_resgates r
LEFT JOIN postos p ON p.id = r.posto_entrega_id
LEFT JOIN usuarios u ON u.id = r.usuario_id
`

func (repo *premioResgatePostgres) CriarTx(ctx context.Context, tx *sql.Tx, r *modelos.PremioResgate) error {
	const q = `
INSERT INTO premio_resgates (
  id, rede_id, premio_id, usuario_id,
  titulo_snapshot, imagem_url_snapshot, valor_moeda, status, prazo_retirada_em
) VALUES (
  $1::uuid, $2::uuid, $3::uuid, $4::uuid,
  $5, NULLIF($6, ''), $7, $8, $9
)
RETURNING criado_em`
	id := strings.TrimSpace(r.ID)
	if id == "" {
		return errors.New("id do resgate obrigatorio")
	}
	return tx.QueryRowContext(
		ctx, q,
		id, strings.TrimSpace(r.IDRede), strings.TrimSpace(r.PremioID), strings.TrimSpace(r.UsuarioID),
		strings.TrimSpace(r.TituloSnapshot), strings.TrimSpace(r.ImagemURLSnapshot),
		r.ValorMoeda, r.Status, r.PrazoRetiradaEm,
	).Scan(&r.CriadoEm)
}

func (repo *premioResgatePostgres) BuscarPorIDNaRede(id, idRede string) (*modelos.PremioResgate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	q := premioResgateSelect + ` WHERE r.id = $1::uuid AND r.rede_id = $2::uuid`
	row := repo.db.QueryRowContext(ctx, q, strings.TrimSpace(id), strings.TrimSpace(idRede))
	out, err := scanPremioResgate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPremioResgateNaoEncontrado
	}
	return out, err
}

func (repo *premioResgatePostgres) ListarPorUsuario(idRede, usuarioID string) ([]*modelos.PremioResgate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	q := premioResgateSelect + `
WHERE r.rede_id = $1::uuid AND r.usuario_id = $2::uuid
ORDER BY r.criado_em DESC
LIMIT 200`
	rows, err := repo.db.QueryContext(ctx, q, strings.TrimSpace(idRede), strings.TrimSpace(usuarioID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lista []*modelos.PremioResgate
	for rows.Next() {
		item, err := scanPremioResgate(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	return lista, rows.Err()
}

func (repo *premioResgatePostgres) ListarPorRede(idRede, status string, limite, offset int) ([]*modelos.PremioResgate, int, error) {
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
	status = strings.TrimSpace(status)
	args := []any{strings.TrimSpace(idRede)}
	where := `WHERE r.rede_id = $1::uuid`
	if status != "" {
		where += ` AND r.status = $2`
		args = append(args, status)
	}
	countQ := `SELECT COUNT(*) FROM premio_resgates r ` + where
	var total int
	if err := repo.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limIdx := len(args) + 1
	offIdx := len(args) + 2
	listQ := premioResgateSelect + where + `
ORDER BY
  CASE r.status WHEN 'AGUARDANDO_RETIRADA' THEN 0 WHEN 'ENTREGUE' THEN 1 ELSE 2 END,
  r.criado_em DESC
LIMIT $` + strconv.Itoa(limIdx) + ` OFFSET $` + strconv.Itoa(offIdx)
	args = append(args, limite, offset)
	rows, err := repo.db.QueryContext(ctx, listQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var lista []*modelos.PremioResgate
	for rows.Next() {
		item, err := scanPremioResgate(rows)
		if err != nil {
			return nil, 0, err
		}
		lista = append(lista, item)
	}
	return lista, total, rows.Err()
}

func (repo *premioResgatePostgres) MarcarEntregue(id, idRede string, postoID *string, operadorUsuarioID, operadorPapel, operadorNome string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
UPDATE premio_resgates SET
  status = 'ENTREGUE',
  entregue_em = NOW(),
  posto_entrega_id = CASE WHEN NULLIF(TRIM($3), '') IS NULL THEN NULL ELSE TRIM($3)::uuid END,
  operador_usuario_id = NULLIF($4, '')::uuid,
  operador_papel = NULLIF($5, ''),
  operador_nome_snapshot = NULLIF($6, '')
WHERE id = $1::uuid
  AND rede_id = $2::uuid
  AND status = 'AGUARDANDO_RETIRADA'`
	postoStr := ""
	if postoID != nil {
		postoStr = strings.TrimSpace(*postoID)
	}
	res, err := repo.db.ExecContext(
		ctx, q,
		strings.TrimSpace(id), strings.TrimSpace(idRede), postoStr,
		strings.TrimSpace(operadorUsuarioID), strings.TrimSpace(operadorPapel), strings.TrimSpace(operadorNome),
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		cur, errB := repo.BuscarPorIDNaRede(id, idRede)
		if errB != nil {
			return errB
		}
		if cur.Status != modelos.PremioResgateAguardandoRetirada {
			return ErrPremioResgateStatus
		}
		return ErrPremioResgateNaoEncontrado
	}
	return nil
}

func (repo *premioResgatePostgres) MarcarCancelado(id, idRede, motivo string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	const q = `
UPDATE premio_resgates SET
  status = 'CANCELADO',
  cancelado_em = NOW(),
  motivo_cancelamento = NULLIF($3, '')
WHERE id = $1::uuid
  AND rede_id = $2::uuid
  AND status = 'AGUARDANDO_RETIRADA'`
	res, err := repo.db.ExecContext(ctx, q, strings.TrimSpace(id), strings.TrimSpace(idRede), strings.TrimSpace(motivo))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		cur, errB := repo.BuscarPorIDNaRede(id, idRede)
		if errB != nil {
			return errB
		}
		if cur.Status != modelos.PremioResgateAguardandoRetirada {
			return ErrPremioResgateStatus
		}
		return ErrPremioResgateNaoEncontrado
	}
	return nil
}
