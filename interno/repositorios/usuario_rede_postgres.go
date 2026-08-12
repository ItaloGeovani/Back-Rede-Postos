package repositorios

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
	"github.com/jackc/pgx/v5/pgconn"
)

// DiagnosticoPushRedeStats contagens para diagnosticar envio FCM (campanhas) por rede.
type DiagnosticoPushRedeStats struct {
	ClientesAtivos      int `json:"clientes_ativos"`
	ClientesComTokenFCM int `json:"clientes_com_token_fcm"`
	TokensDistintos     int `json:"tokens_fcm_distintos"`
}

// ClientePresencaAppItem cliente da rede com dados cadastrais e ultimo heartbeat do app.
type ClientePresencaAppItem struct {
	IDUsuario           string     `json:"id_usuario"`
	NomeCompleto        string     `json:"nome_completo"`
	Email               string     `json:"email"`
	Telefone            string     `json:"telefone"`
	CPF                 string     `json:"cpf"`
	NivelCliente        string     `json:"nivel_cliente"`
	Ativo               bool       `json:"ativo"`
	UltimoAppAcessoEm   *time.Time `json:"ultimo_app_acesso_em,omitempty"`
	UltimoAppPlataforma string     `json:"ultimo_app_plataforma,omitempty"`
	ProvavelmenteOnline bool       `json:"provavelmente_online_agora"`
}

type usuarioRedePostgres struct {
	db *sql.DB
}

func NovoUsuarioRedePostgres(db *sql.DB) *usuarioRedePostgres {
	return &usuarioRedePostgres{db: db}
}

var papeisRedePermitidos = map[string]struct{}{
	"gestor_rede":   {},
	"gerente_posto": {},
	"frentista":     {},
	"cliente":       {},
}

// ErrEmailUsuarioEquipeDuplicado quando ja existe usuario com o mesmo email na rede.
var ErrEmailUsuarioEquipeDuplicado = errors.New("email ja cadastrado nesta rede")

// ErrCodigoAcessoDuplicadoNoPosto quando ja existe frentista com o mesmo codigo no posto.
var ErrCodigoAcessoDuplicadoNoPosto = errors.New("codigo de acesso ja cadastrado neste posto")

// ErrCPFJaCadastradoNaRede quando ja existe usuario com o mesmo CPF na rede.
var ErrCPFJaCadastradoNaRede = errors.New("cpf ja cadastrado nesta rede")

// ErrContaClienteExclusaoNaoAplicada quando o usuario nao e cliente ou nao existe na rede.
var ErrContaClienteExclusaoNaoAplicada = errors.New("conta nao encontrada ou nao e cliente")

// errPresencaParamsInvalidos parametros obrigatorios ausentes em presenca do app.
var errPresencaParamsInvalidos = errors.New("parametros invalidos para presenca")

// ErrPostoNaoPertenceARede quando id_posto nao e da rede informada.
var ErrPostoNaoPertenceARede = errors.New("posto nao pertence a esta rede")

// ErrUsuarioEquipeNaoEncontrado quando o usuario nao existe na rede ou nao e equipe de posto.
var ErrUsuarioEquipeNaoEncontrado = errors.New("usuario da equipe nao encontrado nesta rede")

// ErrUsuarioPainelLoginNaoEncontrado quando nao ha gerente, frentista ou cliente com o email informado.
var ErrUsuarioPainelLoginNaoEncontrado = errors.New("usuario nao encontrado para login no painel")

// SanitizarPapeisFiltro remove valores invalidos; vazio significa "todos os papeis de rede" (exceto super_admin).
func SanitizarPapeisFiltro(papeis []string) []string {
	var out []string
	for _, p := range papeis {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := papeisRedePermitidos[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

func montarFiltroPapelIN(papeis []string, primeiroArg int) (condicao string, args []any) {
	if len(papeis) == 0 {
		return "", nil
	}
	ph := make([]string, len(papeis))
	args = make([]any, len(papeis))
	for i, p := range papeis {
		ph[i] = fmt.Sprintf("$%d", primeiroArg+i)
		args[i] = p
	}
	condicao = " AND u.papel::text IN (" + strings.Join(ph, ",") + ")"
	return condicao, args
}

// ListarPorRedeIDPaginado lista usuarios da rede (exceto super_admin), com filtro opcional de papeis, id_posto e paginacao.
func (r *usuarioRedePostgres) ListarPorRedeIDPaginado(idRede string, limite, offset int, papeisFiltro []string, idPostoFiltro string) ([]*modelos.UsuarioVinculoRede, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	args := []any{strings.TrimSpace(idRede)}
	where := "WHERE u.rede_id = $1::uuid AND u.papel <> 'super_admin'"
	next := 2

	if posto := strings.TrimSpace(idPostoFiltro); posto != "" {
		where += fmt.Sprintf(" AND u.posto_id = $%d::uuid", next)
		args = append(args, posto)
		next++
	}

	filtroPapel, argsPapel := montarFiltroPapelIN(papeisFiltro, next)
	args = append(args, argsPapel...)
	whereSQL := "FROM usuarios u " + where + filtroPapel

	argsCount := append([]any{}, args...)
	queryCount := "SELECT COUNT(*) " + whereSQL
	var total int
	if err := r.db.QueryRowContext(ctx, queryCount, argsCount...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// args ja inclui argsPapel; nao duplicar (evita LIMIT/OFFSET com placeholders errados).
	argsLista := append([]any{}, args...)
	n := len(argsLista)
	lim := fmt.Sprintf("$%d", n+1)
	off := fmt.Sprintf("$%d", n+2)
	argsLista = append(argsLista, limite, offset)

	queryLista := `
SELECT
  u.id::text,
  u.rede_id::text,
  COALESCE(u.posto_id::text, ''),
  u.papel::text,
  u.nome_completo,
  COALESCE(u.email, ''),
  COALESCE(u.codigo_acesso, ''),
  COALESCE(u.telefone, ''),
  u.ativo
` + whereSQL + `
ORDER BY u.papel::text, u.nome_completo
LIMIT ` + lim + ` OFFSET ` + off

	rows, err := r.db.QueryContext(ctx, queryLista, argsLista...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lista []*modelos.UsuarioVinculoRede
	for rows.Next() {
		var u modelos.UsuarioVinculoRede
		if err := rows.Scan(&u.ID, &u.IDRede, &u.IDPosto, &u.Papel, &u.Nome, &u.Email, &u.Codigo, &u.Telefone, &u.Ativo); err != nil {
			return nil, 0, err
		}
		lista = append(lista, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return lista, total, nil
}

// PostoPertenceARede indica se o posto existe e pertence a rede.
func (r *usuarioRedePostgres) PostoPertenceARede(idPosto, idRede string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var um int
	err := r.db.QueryRowContext(ctx, `
SELECT 1 FROM postos WHERE id = $1::uuid AND rede_id = $2::uuid
LIMIT 1`, idPosto, idRede).Scan(&um)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CriarUsuarioEquipe insere gerente de posto ou frentista vinculado a rede e a um posto obrigatorio.
// email pode ser vazio para frentista; codigoAcesso e o codigo de login (obrigatorio para frentista no servico).
func (r *usuarioRedePostgres) CriarUsuarioEquipe(idRede, idPosto, papel, nome, email, senhaHash, telefone, codigoAcesso string) (*modelos.UsuarioVinculoRede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
INSERT INTO usuarios (rede_id, posto_id, papel, nome_completo, email, senha_hash, ativo, telefone, codigo_acesso)
VALUES ($1::uuid, $2::uuid, $3::papel_usuario, $4, NULLIF($5, ''), $6, true, NULLIF($7, ''), NULLIF($8, ''))
RETURNING
  id::text,
  rede_id::text,
  COALESCE(posto_id::text, ''),
  papel::text,
  nome_completo,
  COALESCE(email, ''),
  COALESCE(codigo_acesso, ''),
  COALESCE(telefone, ''),
  ativo`

	var u modelos.UsuarioVinculoRede
	err := r.db.QueryRowContext(
		ctx,
		query,
		idRede,
		strings.TrimSpace(idPosto),
		strings.TrimSpace(papel),
		strings.TrimSpace(nome),
		strings.TrimSpace(email),
		senhaHash,
		strings.TrimSpace(telefone),
		strings.TrimSpace(codigoAcesso),
	).Scan(&u.ID, &u.IDRede, &u.IDPosto, &u.Papel, &u.Nome, &u.Email, &u.Codigo, &u.Telefone, &u.Ativo)
	if err != nil {
		return nil, mapearErroUsuarioEquipePostgres(err)
	}
	return &u, nil
}

// CriarClienteSelfCadastro insere cliente da rede sem posto fixo (posto_id nulo).
func (r *usuarioRedePostgres) CriarClienteSelfCadastro(idRede, nome, email, senhaHash, telefone, cpf string) (*modelos.UsuarioVinculoRede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
INSERT INTO usuarios (rede_id, posto_id, papel, nome_completo, email, senha_hash, ativo, telefone, cpf)
VALUES ($1::uuid, NULL, 'cliente'::papel_usuario, $2, $3, $4, true, NULLIF($5, ''), NULLIF($6, ''))
RETURNING
  id::text,
  rede_id::text,
  COALESCE(posto_id::text, ''),
  papel::text,
  nome_completo,
  COALESCE(email, ''),
  COALESCE(telefone, ''),
  ativo`

	var u modelos.UsuarioVinculoRede
	err := r.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(idRede),
		strings.TrimSpace(nome),
		strings.TrimSpace(email),
		senhaHash,
		strings.TrimSpace(telefone),
		strings.TrimSpace(cpf),
	).Scan(&u.ID, &u.IDRede, &u.IDPosto, &u.Papel, &u.Nome, &u.Email, &u.Telefone, &u.Ativo)
	if err != nil {
		return nil, mapearErroUsuarioEquipePostgres(err)
	}
	return &u, nil
}

// ExcluirContaClientePorID desativa e anonimiza dados pessoais (LGPD / lojas de app).
func (r *usuarioRedePostgres) ExcluirContaClientePorID(idUsuario, idRede string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	idUsuario = strings.TrimSpace(idUsuario)
	idRede = strings.TrimSpace(idRede)
	if idUsuario == "" || idRede == "" {
		return errors.New("ids invalidos")
	}

	emailAnon := fmt.Sprintf("deleted_%s@deleted.local", idUsuario)
	senhaHash := utils.GerarHashSHA256(utils.GerarToken("del"))

	const q = `
UPDATE usuarios SET
  ativo = false,
  nome_completo = 'Conta removida',
  email = $1,
  senha_hash = $2,
  telefone = NULL,
  cpf = NULL,
  atualizado_em = NOW()
WHERE id = $3::uuid AND rede_id = $4::uuid AND papel = 'cliente'::papel_usuario`

	res, err := r.db.ExecContext(ctx, q, emailAnon, senhaHash, idUsuario, idRede)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrContaClienteExclusaoNaoAplicada
	}
	_, _ = r.db.ExecContext(ctx, `DELETE FROM usuario_fcm_tokens WHERE usuario_id = $1::uuid`, idUsuario)
	return nil
}

// UpsertFCMToken regista ou atualiza token FCM; [token] e unico (mesmo aparelho / sessao FCM).
func (r *usuarioRedePostgres) UpsertFCMToken(idUsuario, token, plataforma string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	idUsuario = strings.TrimSpace(idUsuario)
	token = strings.TrimSpace(token)
	plataforma = strings.ToLower(strings.TrimSpace(plataforma))
	if idUsuario == "" || token == "" || len(token) < 20 {
		return errors.New("token fcm invalido")
	}
	if plataforma != "android" && plataforma != "ios" && plataforma != "web" {
		return errors.New("plataforma invalida")
	}

	const q = `
INSERT INTO usuario_fcm_tokens (usuario_id, token, plataforma, atualizado_em)
VALUES ($1::uuid, $2, $3, NOW())
ON CONFLICT (token) DO UPDATE SET
  usuario_id = EXCLUDED.usuario_id,
  plataforma = EXCLUDED.plataforma,
  atualizado_em = NOW()`

	_, err := r.db.ExecContext(ctx, q, idUsuario, token, plataforma)
	return err
}

// ListarTokensFCMPorUsuarioID tokens ativos (mais recente primeiro) para notificações.
func (r *usuarioRedePostgres) ListarTokensFCMPorUsuarioID(idUsuario string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	idUsuario = strings.TrimSpace(idUsuario)
	if idUsuario == "" {
		return nil, errors.New("id usuario invalido")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT token FROM usuario_fcm_tokens
WHERE usuario_id = $1::uuid
ORDER BY atualizado_em DESC
LIMIT 32`, idUsuario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		if strings.TrimSpace(t) != "" {
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

// ListarTokensFCMPorRedeClientesAtivos tokens distintos de clientes ativos da rede (para notificacoes em massa).
func (r *usuarioRedePostgres) ListarTokensFCMPorRedeClientesAtivos(idRede string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, errors.New("id rede invalido")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT t.token
FROM usuario_fcm_tokens t
INNER JOIN usuarios u ON u.id = t.usuario_id
WHERE u.rede_id = $1::uuid
  AND u.papel = 'cliente'::papel_usuario
  AND u.ativo = true
  AND t.token IS NOT NULL
  AND length(trim(t.token)) >= 20`, idRede)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var tok string
		if err := rows.Scan(&tok); err != nil {
			return nil, err
		}
		tok = strings.TrimSpace(tok)
		if tok != "" {
			out = append(out, tok)
		}
	}
	return out, rows.Err()
}

// RemoverTokensFCM apaga tokens da base (ex.: NotRegistered / SenderId mismatch no FCM).
func (r *usuarioRedePostgres) RemoverTokensFCM(tokens []string) (int64, error) {
	limpos := make([]string, 0, len(tokens))
	seen := map[string]struct{}{}
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		limpos = append(limpos, t)
	}
	if len(limpos) == 0 {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	place := make([]string, len(limpos))
	args := make([]any, len(limpos))
	for i, t := range limpos {
		place[i] = fmt.Sprintf("$%d", i+1)
		args[i] = t
	}
	q := `DELETE FROM usuario_fcm_tokens WHERE token IN (` + strings.Join(place, ",") + `)`
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// DiagnosticoPushRede retorna contagens de clientes ativos vs tokens FCM registados na rede.
func (r *usuarioRedePostgres) DiagnosticoPushRede(idRede string) (*DiagnosticoPushRedeStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		return nil, errors.New("id rede invalido")
	}
	const q = `
SELECT
  COALESCE((
    SELECT COUNT(*)::int FROM usuarios u
    WHERE u.rede_id = $1::uuid
      AND u.papel = 'cliente'::papel_usuario
      AND u.ativo = true
  ), 0),
  COALESCE((
    SELECT COUNT(DISTINCT t.usuario_id)::int
    FROM usuario_fcm_tokens t
    INNER JOIN usuarios u ON u.id = t.usuario_id
    WHERE u.rede_id = $1::uuid
      AND u.papel = 'cliente'::papel_usuario
      AND u.ativo = true
      AND t.token IS NOT NULL
      AND length(trim(t.token)) >= 20
  ), 0),
  COALESCE((
    SELECT COUNT(DISTINCT trim(t.token))::int
    FROM usuario_fcm_tokens t
    INNER JOIN usuarios u ON u.id = t.usuario_id
    WHERE u.rede_id = $1::uuid
      AND u.papel = 'cliente'::papel_usuario
      AND u.ativo = true
      AND t.token IS NOT NULL
      AND length(trim(t.token)) >= 20
  ), 0)`

	st := &DiagnosticoPushRedeStats{}
	err := r.db.QueryRowContext(ctx, q, idRede).Scan(&st.ClientesAtivos, &st.ClientesComTokenFCM, &st.TokensDistintos)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func mapearErroUsuarioEquipePostgres(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code == "23505" {
		c := strings.ToLower(pgErr.ConstraintName)
		if strings.Contains(c, "cpf") {
			return ErrCPFJaCadastradoNaRede
		}
		if strings.Contains(c, "codigo_acesso") {
			return ErrCodigoAcessoDuplicadoNoPosto
		}
		return ErrEmailUsuarioEquipeDuplicado
	}
	return err
}

// AtualizarUsuarioEquipe atualiza gerente de posto ou frentista; senhaHash vazio mantem a senha atual.
func (r *usuarioRedePostgres) AtualizarUsuarioEquipe(
	idRede, idUsuario string,
	nome, email, telefone string,
	ativo bool,
	papel, idPosto string,
	senhaHashOuVazio string,
	codigoAcesso string,
) (*modelos.UsuarioVinculoRede, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	idRede = strings.TrimSpace(idRede)
	idUsuario = strings.TrimSpace(idUsuario)
	nome = strings.TrimSpace(nome)
	email = strings.TrimSpace(email)
	telefone = strings.TrimSpace(telefone)
	papel = strings.TrimSpace(papel)
	idPosto = strings.TrimSpace(idPosto)
	senhaHashOuVazio = strings.TrimSpace(senhaHashOuVazio)
	codigoAcesso = strings.TrimSpace(codigoAcesso)

	if idRede == "" || idUsuario == "" || nome == "" || papel == "" || idPosto == "" {
		return nil, ErrDadosInvalidosUsuarioEquipe
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var papelAtual string
	err = tx.QueryRowContext(ctx, `
SELECT papel::text FROM usuarios
WHERE id = $1::uuid AND rede_id = $2::uuid
FOR UPDATE`, idUsuario, idRede).Scan(&papelAtual)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUsuarioEquipeNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	if papelAtual != "gerente_posto" && papelAtual != "frentista" {
		return nil, ErrUsuarioEquipeNaoEncontrado
	}

	if email != "" {
		var duplicado int
		err = tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM usuarios
WHERE rede_id = $1::uuid
  AND email IS NOT NULL AND TRIM(email) <> ''
  AND LOWER(TRIM(email)) = LOWER(TRIM($2))
  AND id <> $3::uuid`, idRede, email, idUsuario).Scan(&duplicado)
		if err != nil {
			return nil, err
		}
		if duplicado > 0 {
			return nil, ErrEmailUsuarioEquipeDuplicado
		}
	}

	ok, err := r.postoPertenceARedeTx(ctx, tx, idPosto, idRede)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPostoNaoPertenceARede
	}

	const query = `
UPDATE usuarios SET
  nome_completo = $2,
  email = NULLIF($3, ''),
  telefone = NULLIF($4, ''),
  ativo = $5,
  papel = $6::papel_usuario,
  posto_id = $7::uuid,
  senha_hash = CASE WHEN $8 = '' THEN senha_hash ELSE $8 END,
  codigo_acesso = NULLIF($9, ''),
  atualizado_em = NOW()
WHERE id = $1::uuid AND rede_id = $10::uuid
RETURNING
  id::text,
  rede_id::text,
  COALESCE(posto_id::text, ''),
  papel::text,
  nome_completo,
  COALESCE(email, ''),
  COALESCE(codigo_acesso, ''),
  COALESCE(telefone, ''),
  ativo`

	var u modelos.UsuarioVinculoRede
	err = tx.QueryRowContext(
		ctx,
		query,
		idUsuario,
		nome,
		email,
		telefone,
		ativo,
		papel,
		idPosto,
		senhaHashOuVazio,
		codigoAcesso,
		idRede,
	).Scan(&u.ID, &u.IDRede, &u.IDPosto, &u.Papel, &u.Nome, &u.Email, &u.Codigo, &u.Telefone, &u.Ativo)
	if err != nil {
		return nil, mapearErroUsuarioEquipePostgres(err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &u, nil
}

// ErrDadosInvalidosUsuarioEquipe parametros obrigatorios ausentes na atualizacao.
var ErrDadosInvalidosUsuarioEquipe = errors.New("dados invalidos para atualizar usuario da equipe")

// UsuarioPainelLogin linha de usuarios (gerente, frentista, cliente) para autenticacao.
type UsuarioPainelLogin struct {
	ID        string
	IDRede    string
	IDPosto   string
	Papel     string
	Nome      string
	SenhaHash string
	Ativo     bool
}

// BuscarPorEmailParaLoginPainel localiza um usuario pelo email (papeis de posto ou cliente).
// Se existir o mesmo email em varias redes, usa o registro mais recente ativo.
func (r *usuarioRedePostgres) BuscarPorEmailParaLoginPainel(email string) (*UsuarioPainelLogin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	email = strings.TrimSpace(email)
	if email == "" {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}

	const query = `
SELECT
  u.id::text,
  u.rede_id::text,
  COALESCE(u.posto_id::text, ''),
  u.papel::text,
  u.nome_completo,
  u.senha_hash,
  u.ativo
FROM usuarios u
WHERE u.papel IN ('gerente_posto', 'frentista', 'cliente')
  AND LOWER(TRIM(u.email)) = LOWER(TRIM($1))
ORDER BY u.ativo DESC, u.criado_em DESC
LIMIT 1`

	var row UsuarioPainelLogin
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&row.ID,
		&row.IDRede,
		&row.IDPosto,
		&row.Papel,
		&row.Nome,
		&row.SenhaHash,
		&row.Ativo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// BuscarPorEmailParaLoginPainelNaRede localiza um usuario pelo email dentro de uma rede especifica.
func (r *usuarioRedePostgres) BuscarPorEmailParaLoginPainelNaRede(idRede, email string) (*UsuarioPainelLogin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	idRede = strings.TrimSpace(idRede)
	email = strings.TrimSpace(email)
	if idRede == "" || email == "" {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}

	const query = `
SELECT
  u.id::text,
  u.rede_id::text,
  COALESCE(u.posto_id::text, ''),
  u.papel::text,
  u.nome_completo,
  u.senha_hash,
  u.ativo
FROM usuarios u
WHERE u.rede_id = $1::uuid
  AND u.papel IN ('gerente_posto', 'frentista', 'cliente')
  AND LOWER(TRIM(u.email)) = LOWER(TRIM($2))
ORDER BY u.ativo DESC, u.criado_em DESC
LIMIT 1`

	var row UsuarioPainelLogin
	err := r.db.QueryRowContext(ctx, query, idRede, email).Scan(
		&row.ID,
		&row.IDRede,
		&row.IDPosto,
		&row.Papel,
		&row.Nome,
		&row.SenhaHash,
		&row.Ativo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListarFrentistasPorCodigoAcesso localiza frentistas pelo codigo de acesso (opcionalmente na rede).
func (r *usuarioRedePostgres) ListarFrentistasPorCodigoAcesso(codigo, idRede string) ([]*UsuarioPainelLogin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	codigo = strings.TrimSpace(codigo)
	idRede = strings.TrimSpace(idRede)
	if codigo == "" {
		return nil, nil
	}

	args := []any{codigo}
	whereRede := ""
	if idRede != "" {
		whereRede = " AND u.rede_id = $2::uuid"
		args = append(args, idRede)
	}

	query := `
SELECT
  u.id::text,
  u.rede_id::text,
  COALESCE(u.posto_id::text, ''),
  u.papel::text,
  u.nome_completo,
  u.senha_hash,
  u.ativo
FROM usuarios u
WHERE u.papel = 'frentista'
  AND u.codigo_acesso IS NOT NULL
  AND TRIM(u.codigo_acesso) <> ''
  AND LOWER(TRIM(u.codigo_acesso)) = LOWER(TRIM($1))` + whereRede + `
ORDER BY u.ativo DESC, u.criado_em DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []*UsuarioPainelLogin
	for rows.Next() {
		var row UsuarioPainelLogin
		if err := rows.Scan(
			&row.ID,
			&row.IDRede,
			&row.IDPosto,
			&row.Papel,
			&row.Nome,
			&row.SenhaHash,
			&row.Ativo,
		); err != nil {
			return nil, err
		}
		lista = append(lista, &row)
	}
	return lista, rows.Err()
}

// BuscarFrentistaAtivoPorCodigoNoPosto localiza frentista ativo pelo codigo no mesmo posto/rede.
func (r *usuarioRedePostgres) BuscarFrentistaAtivoPorCodigoNoPosto(idRede, idPosto, codigo string) (*UsuarioPainelLogin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	idRede = strings.TrimSpace(idRede)
	idPosto = strings.TrimSpace(idPosto)
	codigo = strings.TrimSpace(codigo)
	if idRede == "" || idPosto == "" || codigo == "" {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}

	const query = `
SELECT
  u.id::text,
  u.rede_id::text,
  COALESCE(u.posto_id::text, ''),
  u.papel::text,
  u.nome_completo,
  u.senha_hash,
  u.ativo
FROM usuarios u
WHERE u.rede_id = $1::uuid
  AND u.posto_id = $2::uuid
  AND u.papel = 'frentista'
  AND u.ativo = true
  AND u.codigo_acesso IS NOT NULL
  AND TRIM(u.codigo_acesso) <> ''
  AND LOWER(TRIM(u.codigo_acesso)) = LOWER(TRIM($3))
LIMIT 1`

	var row UsuarioPainelLogin
	err := r.db.QueryRowContext(ctx, query, idRede, idPosto, codigo).Scan(
		&row.ID,
		&row.IDRede,
		&row.IDPosto,
		&row.Papel,
		&row.Nome,
		&row.SenhaHash,
		&row.Ativo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// BuscarFrentistaAtivoPorEmailNoPosto localiza frentista ativo pelo e-mail no mesmo posto/rede.
func (r *usuarioRedePostgres) BuscarFrentistaAtivoPorEmailNoPosto(idRede, idPosto, email string) (*UsuarioPainelLogin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	idRede = strings.TrimSpace(idRede)
	idPosto = strings.TrimSpace(idPosto)
	email = strings.TrimSpace(email)
	if idRede == "" || idPosto == "" || email == "" {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}

	const query = `
SELECT
  u.id::text,
  u.rede_id::text,
  COALESCE(u.posto_id::text, ''),
  u.papel::text,
  u.nome_completo,
  u.senha_hash,
  u.ativo
FROM usuarios u
WHERE u.rede_id = $1::uuid
  AND u.posto_id = $2::uuid
  AND u.papel = 'frentista'
  AND u.ativo = true
  AND u.email IS NOT NULL
  AND TRIM(u.email) <> ''
  AND LOWER(TRIM(u.email)) = LOWER(TRIM($3))
LIMIT 1`

	var row UsuarioPainelLogin
	err := r.db.QueryRowContext(ctx, query, idRede, idPosto, email).Scan(
		&row.ID,
		&row.IDRede,
		&row.IDPosto,
		&row.Papel,
		&row.Nome,
		&row.SenhaHash,
		&row.Ativo,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUsuarioPainelLoginNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// EmailECPFPorUsuarioRede retorna email e CPF (digitos em texto) para o perfil / PIX.
func (r *usuarioRedePostgres) EmailECPFPorUsuarioRede(idUsuario, idRede string) (email string, cpf string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	idUsuario = strings.TrimSpace(idUsuario)
	idRede = strings.TrimSpace(idRede)
	if idUsuario == "" || idRede == "" {
		return "", "", nil
	}
	const q = `
SELECT TRIM(COALESCE(u.email, '')),
  TRIM(COALESCE(u.cpf, ''))
FROM usuarios u
WHERE u.id = $1::uuid AND u.rede_id = $2::uuid`
	err = r.db.QueryRowContext(ctx, q, idUsuario, idRede).Scan(&email, &cpf)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	email = strings.TrimSpace(email)
	cpf = strings.TrimSpace(cpf)
	return email, cpf, nil
}

// DefinirCodigoIndicacao grava o codigo unico (cliente) na rede; falha se duplicar.
func (r *usuarioRedePostgres) DefinirCodigoIndicacao(idUsuario, idRede, codigo string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := strings.ToUpper(strings.TrimSpace(codigo))
	if c == "" {
		return errors.New("codigo vazio")
	}
	const q = `
UPDATE usuarios SET codigo_indicacao = $3, atualizado_em = NOW()
WHERE id = $1::uuid AND rede_id = $2::uuid AND papel = 'cliente'::papel_usuario
`
	res, err := r.db.ExecContext(ctx, q, strings.TrimSpace(idUsuario), strings.TrimSpace(idRede), c)
	if err != nil {
		return mapearErroUsuarioEquipePostgres(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("usuario nao encontrado")
	}
	return nil
}

// ObterNivelCliente codigo do nivel (ex. bronze) para multiplicador de moeda; padrao bronze.
func (r *usuarioRedePostgres) ObterNivelCliente(idUsuario, idRede string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var n sql.NullString
	const q = `SELECT NULLIF(TRIM(LOWER(nivel_cliente)), '') FROM usuarios
WHERE id = $1::uuid AND rede_id = $2::uuid AND papel = 'cliente'::papel_usuario`
	err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(idUsuario), strings.TrimSpace(idRede)).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return "bronze", nil
	}
	if err != nil {
		return "bronze", err
	}
	if !n.Valid || n.String == "" {
		return "bronze", nil
	}
	return n.String, nil
}

// ObterCodigoIndicacao codigo de indicacao (pode vazio se ainda nao atribuido).
func (r *usuarioRedePostgres) ObterCodigoIndicacao(idUsuario, idRede string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var cod sql.NullString
	const q = `SELECT NULLIF(TRIM(codigo_indicacao), '') FROM usuarios
WHERE id = $1::uuid AND rede_id = $2::uuid AND papel = 'cliente'::papel_usuario`
	err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(idUsuario), strings.TrimSpace(idRede)).Scan(&cod)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !cod.Valid {
		return "", nil
	}
	return strings.TrimSpace(cod.String), nil
}

// RegistrarPresencaAppCliente atualiza ultimo_acesso do app para cliente da rede.
func (r *usuarioRedePostgres) RegistrarPresencaAppCliente(idUsuario, idRede, plataforma string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	idUsuario = strings.TrimSpace(idUsuario)
	idRede = strings.TrimSpace(idRede)
	if idUsuario == "" || idRede == "" {
		return errPresencaParamsInvalidos
	}
	pl := strings.TrimSpace(strings.ToLower(plataforma))
	if len(pl) > 48 {
		pl = pl[:48]
	}
	const q = `
UPDATE usuarios
SET ultimo_app_acesso_em = NOW(),
    ultimo_app_plataforma = NULLIF($3::text, '')
WHERE id = $1::uuid AND rede_id = $2::uuid AND papel = 'cliente'::papel_usuario`
	res, err := r.db.ExecContext(ctx, q, idUsuario, idRede, pl)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListarClientesPresencaAppPorRede lista clientes ordenados pelo ultimo app heartbeat (NULL por ultimo).
func (r *usuarioRedePostgres) ListarClientesPresencaAppPorRede(idRede string, limite, minutosOnline int) (totalClientes, totalComPresenca int, itens []ClientePresencaAppItem, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	idRede = strings.TrimSpace(idRede)
	if idRede == "" {
		err = errPresencaParamsInvalidos
		return
	}
	if limite < 1 {
		limite = 200
	}
	if limite > 500 {
		limite = 500
	}
	if minutosOnline < 1 {
		minutosOnline = 15
	}
	if minutosOnline > 180 {
		minutosOnline = 180
	}

	err = r.db.QueryRowContext(ctx, `
SELECT COUNT(*),
       COUNT(*) FILTER (WHERE ultimo_app_acesso_em IS NOT NULL)
FROM usuarios
WHERE rede_id = $1::uuid AND papel = 'cliente'::papel_usuario`, idRede).Scan(&totalClientes, &totalComPresenca)
	if err != nil {
		return
	}

	threshold := time.Now().Add(-time.Duration(minutosOnline) * time.Minute)

	rows, err := r.db.QueryContext(ctx, `
SELECT
  u.id::text,
  u.nome_completo,
  u.email,
  COALESCE(u.telefone, ''),
  COALESCE(u.cpf, ''),
  COALESCE(NULLIF(TRIM(LOWER(u.nivel_cliente)), ''), 'bronze'),
  u.ativo,
  u.ultimo_app_acesso_em,
  COALESCE(NULLIF(TRIM(LOWER(u.ultimo_app_plataforma)), ''), '')
FROM usuarios u
WHERE u.rede_id = $1::uuid AND u.papel = 'cliente'::papel_usuario
ORDER BY u.ultimo_app_acesso_em DESC NULLS LAST
LIMIT $2`, idRede, limite)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var it ClientePresencaAppItem
		var ts sql.NullTime
		var plat string
		if err = rows.Scan(&it.IDUsuario, &it.NomeCompleto, &it.Email, &it.Telefone, &it.CPF, &it.NivelCliente, &it.Ativo, &ts, &plat); err != nil {
			return
		}
		if ts.Valid {
			t := ts.Time.UTC()
			it.UltimoAppAcessoEm = &t
			if !ts.Time.Before(threshold) {
				it.ProvavelmenteOnline = true
			}
		}
		it.UltimoAppPlataforma = plat
		itens = append(itens, it)
	}
	err = rows.Err()
	return
}

// BuscarIdClientePorCodigoIndicacao retorna o id do cliente dono do codigo na rede; vazio se nao existir.
func (r *usuarioRedePostgres) BuscarIdClientePorCodigoIndicacao(idRede, codigo string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c := strings.ToUpper(strings.TrimSpace(codigo))
	if c == "" {
		return "", nil
	}
	var id string
	const q = `
SELECT u.id::text FROM usuarios u
WHERE u.rede_id = $1::uuid AND u.papel = 'cliente'::papel_usuario
  AND upper(trim(codigo_indicacao)) = $2
LIMIT 1`
	err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(idRede), c).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *usuarioRedePostgres) postoPertenceARedeTx(ctx context.Context, tx *sql.Tx, idPosto, idRede string) (bool, error) {
	var um int
	err := tx.QueryRowContext(ctx, `
SELECT 1 FROM postos WHERE id = $1::uuid AND rede_id = $2::uuid
LIMIT 1`, idPosto, idRede).Scan(&um)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
