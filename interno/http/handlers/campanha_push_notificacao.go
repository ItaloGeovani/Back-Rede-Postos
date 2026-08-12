package handlers

import (
	"context"
	"log"
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/notificacoes"
)

// ResultadoPushCampanha explica se o FCM foi agendado (para a resposta HTTP / painel).
type ResultadoPushCampanha struct {
	Disparado bool   `json:"disparado"`
	Motivo    string `json:"motivo"`
}

func avaliarPushCampanha(c *modelos.Campanha) (ok bool, motivo string) {
	if c == nil {
		return false, "campanha nula"
	}
	if c.Status != modelos.StatusCampanhaAtiva {
		return false, "status nao e ATIVA (criou/deixou como " + string(c.Status) + ")"
	}
	if !c.ValidaNoApp {
		return false, "valida_no_app=false (campanha so no posto fisico nao notifica o app)"
	}
	if strings.TrimSpace(c.IDRede) == "" {
		return false, "id_rede vazio"
	}
	return true, "agendado"
}

// notificarClientesPushNovaCampanha envia FCM (assincrono) quando a campanha fica ativa e valida no app.
func (h *Handlers) notificarClientesPushNovaCampanha(c *modelos.Campanha) ResultadoPushCampanha {
	ok, motivo := avaliarPushCampanha(c)
	log.Printf("fcm campanha: verificar push campanha_id=%s status=%s valida_no_app=%v id_rede=%s → %s",
		safeID(c), statusCampanha(c), validaApp(c), idRedeCampanha(c), motivo)
	if !ok {
		return ResultadoPushCampanha{Disparado: false, Motivo: motivo}
	}
	idRede := strings.TrimSpace(c.IDRede)
	campanhaID := c.ID
	titulo := tituloCampanhaPush(c)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if strings.TrimSpace(h.cfg.FcmCaminhoContaServico) == "" {
			log.Printf("fcm campanha: nao enviado (defina FCM_SA no .env e reinicie o servidor)")
			return
		}
		tokens, err := h.usuarioRedeService.ListarTokensFCMClientesRede(idRede)
		if err != nil {
			log.Printf("fcm campanha: listar tokens rede %s: %v", idRede, err)
			return
		}
		if len(tokens) == 0 {
			log.Printf("fcm campanha: nao enviado (0 tokens FCM para clientes ativos desta rede id_rede=%s; app cliente precisa de login e POST /v1/eu/push/fcm na mesma rede)", idRede)
			return
		}
		log.Printf("fcm campanha: a enviar para %d token(s) rede=%s campanha=%s", len(tokens), idRede, campanhaID)
		invalidos := notificacoes.EnviarNovaCampanhaNoApp(ctx, h.cfg.FcmCaminhoContaServico, tokens, campanhaID, titulo, idRede)
		if len(invalidos) > 0 {
			n, err := h.usuarioRedeService.RemoverTokensFCM(invalidos)
			if err != nil {
				log.Printf("fcm campanha: limpar tokens invalidos: %v", err)
			} else {
				log.Printf("fcm campanha: removidos %d token(s) invalidos (SenderId mismatch/NotRegistered)", n)
			}
		}
	}()

	return ResultadoPushCampanha{
		Disparado: true,
		Motivo:    "envio FCM agendado para clientes com token na rede",
	}
}

func safeID(c *modelos.Campanha) string {
	if c == nil {
		return ""
	}
	return c.ID
}

func statusCampanha(c *modelos.Campanha) modelos.StatusCampanha {
	if c == nil {
		return ""
	}
	return c.Status
}

func validaApp(c *modelos.Campanha) bool {
	return c != nil && c.ValidaNoApp
}

func idRedeCampanha(c *modelos.Campanha) string {
	if c == nil {
		return ""
	}
	return c.IDRede
}

func tituloCampanhaPush(c *modelos.Campanha) string {
	if c == nil {
		return "Nova promocao"
	}
	for _, t := range []string{c.TituloExibicao, c.Titulo, c.Nome} {
		if s := strings.TrimSpace(t); s != "" {
			return s
		}
	}
	return "Nova promocao"
}

// campanhaAgoraAtivaNoApp: o estado novo e ATIVA+app, e antes nao era (ex.: pausa, rascunho, ou desligou app e religou).
func campanhaAgoraAtivaNoApp(antiga, nova *modelos.Campanha) bool {
	if antiga == nil || nova == nil {
		return false
	}
	if nova.Status != modelos.StatusCampanhaAtiva || !nova.ValidaNoApp {
		return false
	}
	if antiga.Status == modelos.StatusCampanhaAtiva && antiga.ValidaNoApp {
		return false
	}
	return true
}

// notificarClientesSeCampanhaAtivada dispara o mesmo FCM de criacao ao reativar/editar para ATIVA.
func (h *Handlers) notificarClientesSeCampanhaAtivada(antiga, nova *modelos.Campanha) ResultadoPushCampanha {
	if !campanhaAgoraAtivaNoApp(antiga, nova) {
		motivo := "sem transicao para ATIVA+app"
		if antiga != nil && nova != nil {
			motivo = "sem transicao para ATIVA+app (antes=" + string(antiga.Status) +
				"/app=" + boolStr(antiga.ValidaNoApp) +
				" depois=" + string(nova.Status) +
				"/app=" + boolStr(nova.ValidaNoApp) +
				"); push so ao criar ATIVA ou ao mudar de pausada/rascunho para ATIVA"
		}
		log.Printf("fcm campanha: skip ativacao — %s", motivo)
		return ResultadoPushCampanha{Disparado: false, Motivo: motivo}
	}
	return h.notificarClientesPushNovaCampanha(nova)
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
