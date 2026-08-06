package handlers

import (
	"strings"
	"time"

	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
)

func (h *Handlers) registrarEventoCampanhaCriada(c *modelos.Campanha) {
	if h == nil || h.eventosSvc == nil || c == nil {
		return
	}
	titulo := strings.TrimSpace(c.TituloExibicao)
	if titulo == "" {
		titulo = strings.TrimSpace(c.Titulo)
	}
	if titulo == "" {
		titulo = strings.TrimSpace(c.Nome)
	}
	idEnt := strings.TrimSpace(c.ID)
	var entPtr *string
	if idEnt != "" {
		entPtr = &idEnt
	}
	var postoPtr *string
	if p := strings.TrimSpace(c.IDPosto); p != "" {
		postoPtr = &p
	}
	h.eventosSvc.Registrar(servicos.RegistrarEventoInput{
		IDRede:       c.IDRede,
		IDPosto:      postoPtr,
		TipoEvento:   modelos.EventoCampanhaCriada,
		EntidadeTipo: "campanha",
		IDEntidade:   entPtr,
		Titulo:       titulo,
		Status:       string(c.Status),
		DataHora:     time.Now().Format("02/01/2006 15:04"),
		Payload: map[string]any{
			"campanha_id": c.ID,
			"nome":        c.Nome,
			"titulo":      titulo,
			"status":      c.Status,
		},
	})
}

func (h *Handlers) registrarEventoCampanhaAtivada(antiga, nova *modelos.Campanha) {
	if h == nil || h.eventosSvc == nil || antiga == nil || nova == nil {
		return
	}
	if nova.Status != modelos.StatusCampanhaAtiva {
		return
	}
	if antiga.Status == modelos.StatusCampanhaAtiva {
		return
	}
	titulo := strings.TrimSpace(nova.TituloExibicao)
	if titulo == "" {
		titulo = strings.TrimSpace(nova.Titulo)
	}
	if titulo == "" {
		titulo = strings.TrimSpace(nova.Nome)
	}
	idEnt := strings.TrimSpace(nova.ID)
	var entPtr *string
	if idEnt != "" {
		entPtr = &idEnt
	}
	var postoPtr *string
	if p := strings.TrimSpace(nova.IDPosto); p != "" {
		postoPtr = &p
	}
	h.eventosSvc.Registrar(servicos.RegistrarEventoInput{
		IDRede:       nova.IDRede,
		IDPosto:      postoPtr,
		TipoEvento:   modelos.EventoCampanhaAtivada,
		EntidadeTipo: "campanha",
		IDEntidade:   entPtr,
		Titulo:       titulo,
		Status:       string(nova.Status),
		DataHora:     time.Now().Format("02/01/2006 15:04"),
		Payload: map[string]any{
			"campanha_id": nova.ID,
			"nome":        nova.Nome,
			"titulo":      titulo,
			"status":      nova.Status,
			"status_ant":  antiga.Status,
		},
	})
}
