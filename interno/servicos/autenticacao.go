package servicos

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/utils"
)

var ErrTokenInvalido = errors.New("token invalido")
var ErrSessaoExpirada = errors.New("sessao expirada; faca login novamente")

type Autenticador interface {
	ValidarToken(token string) (*modelos.UsuarioSessao, error)
	CriarSessao(usuario *modelos.UsuarioSessao) string
	RevogarToken(token string)
	// RevogarSessoesPorPapel invalida tok_* do papel (ex.: cliente). idRede vazio = todas as redes.
	RevogarSessoesPorPapel(papel modelos.Papel, idRede string) (int64, error)
}

// entradaSessao: se ate.IsZero(), a entrada em memoria nao expira (modo sem Postgres).
type entradaSessao struct {
	u   *modelos.UsuarioSessao
	ate time.Time
}

type autenticadorToken struct {
	tokenPadrao   string
	db            *sql.DB
	duracaoSessao time.Duration
	mu            sync.RWMutex
	sessoes       map[string]entradaSessao
}

// NovoAutenticadorToken: apenas RAM (desenvolvimento / testes); tok_* nao expiram em memoria.
func NovoAutenticadorToken(tokenPadrao string) Autenticador {
	return &autenticadorToken{
		tokenPadrao: tokenPadrao,
		sessoes:     make(map[string]entradaSessao),
	}
}

// NovoAutenticadorTokenComPersistencia: grava tok_* no Postgres; sobrevive a restarts; expiracao de [dur].
func NovoAutenticadorTokenComPersistencia(db *sql.DB, tokenPadrao string, dur time.Duration) Autenticador {
	if dur < time.Hour {
		dur = 180 * 24 * time.Hour
	}
	return &autenticadorToken{
		tokenPadrao:   tokenPadrao,
		db:            db,
		duracaoSessao: dur,
		sessoes:       make(map[string]entradaSessao),
	}
}

func (a *autenticadorToken) ValidarToken(token string) (*modelos.UsuarioSessao, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrTokenInvalido
	}
	if strings.HasPrefix(token, "tok_") {
		return a.validarSessaoToken(token)
	}

	switch token {
	case a.tokenPadrao, "dev-super-admin":
		return &modelos.UsuarioSessao{
			IDUsuario:    "usuario-dev-super-admin",
			NomeCompleto: "Administrador Global Dev",
			IDRede:       "rede-dev-001",
			Papel:        modelos.PapelSuperAdmin,
		}, nil
	case "dev-gestor":
		return &modelos.UsuarioSessao{
			IDUsuario:    "usuario-dev-gestor",
			NomeCompleto: "Gestor de Rede Dev",
			IDRede:       "rede-dev-001",
			Papel:        modelos.PapelGestorRede,
		}, nil
	case "dev-frentista":
		return &modelos.UsuarioSessao{
			IDUsuario:    "usuario-dev-frentista",
			NomeCompleto: "Frentista Dev",
			IDRede:       "rede-dev-001",
			Papel:        modelos.PapelFrentista,
		}, nil
	default:
		return nil, ErrTokenInvalido
	}
}

func (a *autenticadorToken) validarSessaoToken(token string) (*modelos.UsuarioSessao, error) {
	a.purgarMemSeExpirado(token)

	a.mu.RLock()
	ent, ok := a.sessoes[token]
	a.mu.RUnlock()
	if ok {
		if !ent.ate.IsZero() && time.Now().After(ent.ate) {
			a.mu.Lock()
			delete(a.sessoes, token)
			a.mu.Unlock()
			if a.db != nil {
				_, _ = a.dbExec(`DELETE FROM sessao_api WHERE token = $1`, token)
			}
		} else {
			if ent.u == nil {
				return nil, ErrSessaoExpirada
			}
			c := *ent.u
			return &c, nil
		}
	}

	if a.db == nil {
		return nil, ErrSessaoExpirada
	}
	u, expira, err := a.carregarSessaoDB(context.Background(), token)
	if err != nil {
		log.Printf("sessao_api: leitura: %v", err)
		return nil, ErrSessaoExpirada
	}
	if u == nil {
		return nil, ErrSessaoExpirada
	}
	a.mu.Lock()
	a.sessoes[token] = entradaSessao{u: u, ate: expira}
	a.mu.Unlock()
	c := *u
	return &c, nil
}

func (a *autenticadorToken) carregarSessaoDB(ctx context.Context, token string) (*modelos.UsuarioSessao, time.Time, error) {
	var (
		uid, idRede, idPosto, nome, papel string
		expira                            time.Time
	)
	err := a.db.QueryRowContext(ctx, `
SELECT usuario_id, id_rede, id_posto, nome_completo, papel, expira_em
FROM sessao_api WHERE token = $1 AND expira_em > now()
`, token).Scan(&uid, &idRede, &idPosto, &nome, &papel, &expira)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	return &modelos.UsuarioSessao{
		IDUsuario:    uid,
		NomeCompleto: nome,
		IDRede:       idRede,
		IDPosto:      idPosto,
		Papel:        modelos.Papel(papel),
	}, expira, nil
}

func (a *autenticadorToken) purgarMemSeExpirado(token string) {
	a.mu.RLock()
	ent, ok := a.sessoes[token]
	a.mu.RUnlock()
	if !ok || ent.ate.IsZero() {
		return
	}
	if !time.Now().After(ent.ate) {
		return
	}
	a.mu.Lock()
	delete(a.sessoes, token)
	a.mu.Unlock()
}

func (a *autenticadorToken) dbExec(q string, args ...any) (sql.Result, error) {
	if a.db == nil {
		return nil, nil
	}
	return a.db.ExecContext(context.Background(), q, args...)
}

func (a *autenticadorToken) CriarSessao(usuario *modelos.UsuarioSessao) string {
	token := utils.GerarToken("tok")
	copia := *usuario
	u := &copia
	var memAte time.Time
	if a.db != nil {
		memAte = time.Now().Add(a.duracaoSessao)
		_, err := a.dbExec(`
INSERT INTO sessao_api (token, usuario_id, id_rede, id_posto, nome_completo, papel, expira_em, criado_em)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
`, token, u.IDUsuario, u.IDRede, u.IDPosto, u.NomeCompleto, string(u.Papel), memAte)
		if err != nil {
			log.Printf("sessao_api: insercao: %v (sessao so em memoria ate reinicio)", err)
		}
	} else {
		memAte = time.Time{}
	}
	a.mu.Lock()
	a.sessoes[token] = entradaSessao{u: u, ate: memAte}
	a.mu.Unlock()
	return token
}

func (a *autenticadorToken) RevogarToken(token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	if a.db != nil {
		_, _ = a.dbExec(`DELETE FROM sessao_api WHERE token = $1`, token)
	}
	a.mu.Lock()
	delete(a.sessoes, token)
	a.mu.Unlock()
}

// RevogarSessoesPorPapel apaga sessões no Postgres e remove da memória as do mesmo papel/rede.
func (a *autenticadorToken) RevogarSessoesPorPapel(papel modelos.Papel, idRede string) (int64, error) {
	papelStr := strings.TrimSpace(string(papel))
	if papelStr == "" {
		return 0, errors.New("papel obrigatorio")
	}
	idRede = strings.TrimSpace(idRede)

	var n int64
	if a.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		var (
			res sql.Result
			err error
		)
		if idRede == "" {
			res, err = a.db.ExecContext(ctx, `DELETE FROM sessao_api WHERE papel = $1`, papelStr)
		} else {
			res, err = a.db.ExecContext(ctx, `DELETE FROM sessao_api WHERE papel = $1 AND id_rede = $2`, papelStr, idRede)
		}
		if err != nil {
			return 0, err
		}
		n, _ = res.RowsAffected()
	}

	a.mu.Lock()
	for tok, ent := range a.sessoes {
		if ent.u == nil || ent.u.Papel != papel {
			continue
		}
		if idRede != "" && strings.TrimSpace(ent.u.IDRede) != idRede {
			continue
		}
		delete(a.sessoes, tok)
	}
	a.mu.Unlock()

	log.Printf("sessao_api: revogadas papel=%s id_rede=%q removidas_db=%d", papelStr, idRede, n)
	return n, nil
}
