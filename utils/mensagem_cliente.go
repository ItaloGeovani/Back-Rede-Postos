package utils

import (
	"strings"
)

// MsgFormularioIncompleto mensagem genérica quando não dá para apontar o campo.
const MsgFormularioIncompleto = "Verifique os dados informados e tente novamente."

// MensagemDecodeJSON traduz falha de JSON para texto legível (nunca "payload invalido").
func MensagemDecodeJSON(err error) string {
	if err == nil {
		return MsgFormularioIncompleto
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unknown field") {
		return "Alguns dados enviados não são reconhecidos. Atualize o aplicativo e tente novamente."
	}
	return MsgFormularioIncompleto
}

// MensagemParaCliente remove jargão técnico (payload, dados invalidos) e deixa a causa clara.
func MensagemParaCliente(err error) string {
	if err == nil {
		return MsgFormularioIncompleto
	}
	return MensagemParaClienteTexto(err.Error())
}

// MensagemParaClienteTexto humaniza uma mensagem de erro já em texto.
func MensagemParaClienteTexto(raw string) string {
	msg := strings.TrimSpace(raw)
	if msg == "" {
		return MsgFormularioIncompleto
	}
	low := strings.ToLower(msg)

	switch low {
	case "payload invalido", "payload inválido",
		"dados invalidos", "dados inválidos",
		"json invalido", "json inválido",
		"eof", "unexpected eof":
		return MsgFormularioIncompleto
	}

	for _, p := range []string{
		"dados invalidos: ",
		"dados inválidos: ",
		"payload invalido: ",
		"payload inválido: ",
	} {
		if strings.HasPrefix(low, p) {
			msg = strings.TrimSpace(msg[len(p):])
			low = strings.ToLower(msg)
			break
		}
	}

	if low == "" || low == "dados invalidos" || low == "dados inválidos" ||
		low == "payload invalido" || low == "payload inválido" {
		return MsgFormularioIncompleto
	}

	if strings.Contains(low, "unknown field") {
		return "Alguns dados enviados não são reconhecidos. Atualize o aplicativo e tente novamente."
	}

	switch {
	case strings.Contains(low, "senha e confirmar"):
		return "A senha e a confirmação não são iguais."
	case strings.Contains(low, "senha deve ter no minimo 2"),
		strings.Contains(low, "senha deve ter no mínimo 2"):
		return "A senha deve ter no mínimo 2 caracteres."
	case strings.Contains(low, "senha deve ter no minimo 6"),
		strings.Contains(low, "senha deve ter no mínimo 6"):
		return "A senha deve ter no mínimo 6 caracteres."
	case strings.Contains(low, "senha deve ter no minimo"),
		strings.Contains(low, "senha deve ter no mínimo"):
		return "A senha está muito curta."
	case strings.Contains(low, "informe a senha"):
		return "Informe a senha."
	case strings.Contains(low, "usuario deve ter") || strings.Contains(low, "usuário deve ter"):
		return "O usuário deve ter no mínimo 4 caracteres, apenas letras e números (sem acento, espaço ou símbolos)."
	case strings.Contains(low, "codigo de acesso obrigatorio") || strings.Contains(low, "código de acesso obrigatório"):
		return "Informe o código de acesso."
	case strings.Contains(low, "email obrigatorio") || strings.Contains(low, "e-mail obrigatório"):
		return "Informe o e-mail."
	case strings.Contains(low, "e-mail invalido") || strings.Contains(low, "email invalido") ||
		strings.Contains(low, "e-mail inválido") || strings.Contains(low, "email inválido"):
		return "Informe um e-mail válido."
	case strings.Contains(low, "cpf invalido") || strings.Contains(low, "cpf inválido"):
		return "Informe um CPF válido."
	case strings.Contains(low, "usuario ja cadastrado") || strings.Contains(low, "usuário já cadastrado"):
		return "Este usuário já está cadastrado nesta rede."
	case strings.Contains(low, "rede não identificada") || strings.Contains(low, "rede nao identificada"):
		return "Não foi possível identificar a rede. Atualize o aplicativo e tente novamente."
	case strings.Contains(low, "nome") && strings.Contains(low, "obrigat"):
		return "Informe o nome completo."
	}

	if strings.Contains(low, "payload") {
		return MsgFormularioIncompleto
	}

	return capitalizarFrase(msg)
}

func capitalizarFrase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return MsgFormularioIncompleto
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 'a' + 'A'
	}
	return string(r)
}
