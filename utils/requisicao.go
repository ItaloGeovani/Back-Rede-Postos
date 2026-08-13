package utils

import (
	"encoding/json"
	"net/http"
)

func DecodificarJSON(r *http.Request, destino any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	// Campos extras são ignorados: apps/painéis evoluem e DisallowUnknownFields
	// gerava "payload invalido" sem explicar o motivo ao usuário.
	return decoder.Decode(destino)
}
