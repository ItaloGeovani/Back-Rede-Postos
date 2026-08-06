package servicos

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/notificacoes"
	"gaspass-servidor/interno/repositorios"
)

// ServicoEventosOperacionais persiste logs e dispara WhatsApp async.
type ServicoEventosOperacionais struct {
	repo  repositorios.EventosOperacionaisRepositorio
	waCfg repositorios.WhatsAppNotificacoesRepositorio
	wa    *notificacoes.WhatsAppNotifier
	rede  repositorios.RedeRepositorio
	posto interface {
		BuscarPorIDNaRede(idPosto, idRede string) (*modelos.Posto, error)
	}
	db *sql.DB
}

func NovoServicoEventosOperacionais(
	db *sql.DB,
	repo repositorios.EventosOperacionaisRepositorio,
	waCfg repositorios.WhatsAppNotificacoesRepositorio,
	wa *notificacoes.WhatsAppNotifier,
	rede repositorios.RedeRepositorio,
	posto interface {
		BuscarPorIDNaRede(idPosto, idRede string) (*modelos.Posto, error)
	},
) *ServicoEventosOperacionais {
	return &ServicoEventosOperacionais{
		db: db, repo: repo, waCfg: waCfg, wa: wa, rede: rede, posto: posto,
	}
}

type RegistrarEventoInput struct {
	IDRede       string
	IDPosto      *string
	TipoEvento   string
	EntidadeTipo string
	IDEntidade   *string
	Titulo       string
	Payload      map[string]any
	// Template
	Valor    string
	Quem     string
	Meio     string
	Status   string
	Codigo   string
	Extra    string // operador etc.
	DataHora string
}

func (s *ServicoEventosOperacionais) Registrar(in RegistrarEventoInput) {
	if s == nil || s.repo == nil {
		return
	}
	idRede := strings.TrimSpace(in.IDRede)
	tipo := strings.TrimSpace(in.TipoEvento)
	if idRede == "" || tipo == "" {
		return
	}
	payload := in.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	if in.Valor != "" {
		payload["valor"] = in.Valor
	}
	if in.Quem != "" {
		payload["quem"] = in.Quem
	}
	if in.Meio != "" {
		payload["meio"] = in.Meio
	}
	if in.Status != "" {
		payload["status"] = in.Status
	}
	if in.Codigo != "" {
		payload["codigo"] = in.Codigo
	}
	if in.Extra != "" {
		payload["extra"] = in.Extra
	}
	ev := &modelos.EventoOperacional{
		IDRede:       idRede,
		IDPosto:      in.IDPosto,
		TipoEvento:   tipo,
		EntidadeTipo: strings.TrimSpace(in.EntidadeTipo),
		IDEntidade:   in.IDEntidade,
		Titulo:       strings.TrimSpace(in.Titulo),
		Payload:      notificacoes.PayloadMap(payload),
	}
	if err := s.repo.Inserir(ev); err != nil {
		log.Printf("evento operacional insert falhou tipo=%s rede=%s: %v", tipo, idRede, err)
		return
	}
	cab := s.cabecalho(idRede, in.IDPosto)
	dados := notificacoes.WhatsAppTemplateDados{
		Cabecalho: cab,
		Valor:     in.Valor,
		Quem:      in.Quem,
		DataHora:  in.DataHora,
		Meio:      in.Meio,
		Status:    in.Status,
		Codigo:    in.Codigo,
		Titulo:    in.Titulo,
		Extra:     in.Extra,
	}
	if s.wa != nil {
		s.wa.NotificarAsync(ev, cab, dados)
	}
}

func (s *ServicoEventosOperacionais) Listar(idRede string, limite, offset int) ([]*modelos.EventoOperacional, int, error) {
	if s == nil || s.repo == nil {
		return nil, 0, fmt.Errorf("servico indisponivel")
	}
	return s.repo.ListarPorRede(idRede, limite, offset)
}

func (s *ServicoEventosOperacionais) BuscarConfigWhatsApp(idRede string) (*modelos.RedeWhatsAppNotificacoes, error) {
	if s == nil || s.waCfg == nil {
		return nil, fmt.Errorf("servico indisponivel")
	}
	return s.waCfg.BuscarPorRede(idRede)
}

func (s *ServicoEventosOperacionais) SalvarConfigWhatsApp(c *modelos.RedeWhatsAppNotificacoes) error {
	if s == nil || s.waCfg == nil || c == nil {
		return fmt.Errorf("servico indisponivel")
	}
	return s.waCfg.Upsert(c)
}

func (s *ServicoEventosOperacionais) EnviarTesteWhatsApp(ctx context.Context, idRede string) error {
	if s == nil || s.wa == nil {
		return fmt.Errorf("WhatsApp desligado no servidor (EVOLUTION_GO_BASE_URL)")
	}
	return s.wa.EnviarTeste(ctx, idRede)
}

func (s *ServicoEventosOperacionais) cabecalho(idRede string, idPosto *string) string {
	if idPosto != nil && strings.TrimSpace(*idPosto) != "" && s.posto != nil {
		if p, err := s.posto.BuscarPorIDNaRede(strings.TrimSpace(*idPosto), idRede); err == nil && p != nil {
			n := strings.TrimSpace(p.NomeFantasia)
			if n == "" {
				n = strings.TrimSpace(p.Nome)
			}
			if n != "" {
				return n
			}
		}
	}
	if s.rede != nil {
		if r, err := s.rede.BuscarPorID(idRede); err == nil && r != nil {
			n := strings.TrimSpace(r.NomeFantasia)
			if n == "" {
				n = strings.TrimSpace(r.RazaoSocial)
			}
			if n != "" {
				return n
			}
		}
	}
	return "REDE"
}

func (s *ServicoEventosOperacionais) NomeUsuario(idUsuario, idRede string) string {
	idUsuario = strings.TrimSpace(idUsuario)
	idRede = strings.TrimSpace(idRede)
	if s == nil || s.db == nil || idUsuario == "" {
		return idUsuario
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var nome sql.NullString
	err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(NULLIF(TRIM(nome_completo), ''), NULLIF(TRIM(email), ''), id::text)
FROM usuarios WHERE id = $1::uuid AND ($2 = '' OR rede_id = $2::uuid)
LIMIT 1`, idUsuario, idRede).Scan(&nome)
	if err != nil || !nome.Valid || nome.String == "" {
		return idUsuario
	}
	return nome.String
}

func FormatValorBR(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func MascararToken(tok string) string {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return ""
	}
	if len(tok) <= 8 {
		return "****"
	}
	return tok[:4] + "…" + tok[len(tok)-4:]
}

// PayloadRaw helper for JSON responses.
func PayloadAsMap(raw json.RawMessage) map[string]any {
	var m map[string]any
	if len(raw) == 0 {
		return m
	}
	_ = json.Unmarshal(raw, &m)
	return m
}
