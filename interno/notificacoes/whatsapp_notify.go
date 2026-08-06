package notificacoes

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/repositorios"
)

type WhatsAppNotifier struct {
	BaseURL string
	CfgRepo repositorios.WhatsAppNotificacoesRepositorio
}

func (n *WhatsAppNotifier) NotificarAsync(evento *modelos.EventoOperacional, cabecalho string, dados WhatsAppTemplateDados) {
	if n == nil || evento == nil || strings.TrimSpace(n.BaseURL) == "" || n.CfgRepo == nil {
		return
	}
	ev := *evento
	cab := cabecalho
	d := dados
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("whatsapp notify panic: %v", r)
			}
		}()
		cfg, err := n.CfgRepo.BuscarPorRede(ev.IDRede)
		if err != nil || cfg == nil {
			return
		}
		if !cfg.FlagParaTipo(ev.TipoEvento) {
			return
		}
		token := strings.TrimSpace(cfg.InstanceToken)
		jid := strings.TrimSpace(cfg.GroupJID)
		if token == "" || jid == "" {
			return
		}
		if strings.TrimSpace(d.Cabecalho) == "" {
			d.Cabecalho = cab
		}
		if strings.TrimSpace(d.DataHora) == "" && !ev.CriadoEm.IsZero() {
			d.DataHora = ev.CriadoEm.In(time.Local).Format("02/01/2006 15:04")
		}
		if strings.TrimSpace(d.Titulo) == "" {
			d.Titulo = ev.Titulo
		}
		texto := RenderWhatsAppTemplate(ev.TipoEvento, VarianteTemplate(ev.ID), d)
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if err := EvolutionSendText(ctx, n.BaseURL, token, jid, texto); err != nil {
			log.Printf("whatsapp evolution falhou rede=%s evento=%s: %v", ev.IDRede, ev.ID, err)
		}
	}()
}

func (n *WhatsAppNotifier) EnviarTeste(ctx context.Context, idRede string) error {
	if n == nil || strings.TrimSpace(n.BaseURL) == "" {
		return errWhatsAppDesligado()
	}
	cfg, err := n.CfgRepo.BuscarPorRede(idRede)
	if err != nil {
		return err
	}
	token := strings.TrimSpace(cfg.InstanceToken)
	jid := strings.TrimSpace(cfg.GroupJID)
	if token == "" || jid == "" {
		return errWhatsAppIncompleto()
	}
	texto := "== [ TESTE ] =====\n*Mensagem de teste*\nLogs operacionais WhatsApp OK.\n" +
		time.Now().Format("02/01/2006 15:04") + "\n=================="
	return EvolutionSendText(ctx, n.BaseURL, token, jid, texto)
}

type errWA string

func (e errWA) Error() string { return string(e) }

func errWhatsAppDesligado() error {
	return errWA("WhatsApp desligado no servidor (EVOLUTION_GO_BASE_URL)")
}

func errWhatsAppIncompleto() error {
	return errWA("configure instance_token e group_jid antes do teste")
}

// PayloadMap helpers para montar payload JSON.
func PayloadMap(m map[string]any) json.RawMessage {
	b, err := json.Marshal(m)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}
