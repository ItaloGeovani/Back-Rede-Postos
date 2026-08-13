package utils

import (
	"encoding/json"
	"net/http"
)

func ResponderJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func ResponderErro(w http.ResponseWriter, status int, mensagem string) {
	ResponderJSON(w, status, map[string]string{
		"erro": MensagemParaClienteTexto(mensagem),
	})
}

// ResponderErroCliente responde com mensagem já humanizada a partir de um error.
func ResponderErroCliente(w http.ResponseWriter, status int, err error) {
	ResponderErro(w, status, MensagemParaCliente(err))
}
