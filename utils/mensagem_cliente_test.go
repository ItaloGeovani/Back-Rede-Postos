package utils

import (
	"testing"
)

func TestMensagemParaClienteTexto(t *testing.T) {
	cases := map[string]string{
		"payload invalido": "Verifique os dados informados e tente novamente.",
		"dados invalidos":  "Verifique os dados informados e tente novamente.",
		"dados invalidos: senha e confirmar_senha devem ser iguais": "A senha e a confirmação não são iguais.",
		"dados invalidos: senha deve ter no minimo 6 caracteres":    "A senha deve ter no mínimo 6 caracteres.",
		"dados invalidos: e-mail invalido":                         "Informe um e-mail válido.",
		"dados invalidos: cpf invalido":                            "Informe um CPF válido.",
		"dados invalidos: usuario deve ter no minimo 4 caracteres, apenas letras minusculas e numeros (sem acento, espaco ou simbolos)": "O usuário deve ter no mínimo 4 caracteres, apenas letras e números (sem acento, espaço ou símbolos).",
	}
	for in, want := range cases {
		got := MensagemParaClienteTexto(in)
		if got != want {
			t.Fatalf("in=%q\ngot=%q\nwant=%q", in, got, want)
		}
	}
}
