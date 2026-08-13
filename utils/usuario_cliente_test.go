package utils

import "testing"

func TestNormalizarUsuarioCliente(t *testing.T) {
	if got := NormalizarUsuarioCliente(" Joao_1 "); got != "joao1" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizarUsuarioCliente("ABC12"); got != "abc12" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizarUsuarioCliente("João"); got != "joo" {
		t.Fatalf("acento deve ser removido, got %q", got)
	}
}

func TestValidarUsuarioCliente(t *testing.T) {
	u, err := ValidarUsuarioCliente("joao")
	if err != nil || u != "joao" {
		t.Fatalf("esperado joao: %q %v", u, err)
	}
	if _, err := ValidarUsuarioCliente("abc"); err == nil {
		t.Fatal("esperado erro para curto")
	}
	if _, err := ValidarUsuarioCliente("joao_1"); err == nil {
		t.Fatal("esperado erro para underscore")
	}
	u, err = ValidarUsuarioCliente("Joao1")
	if err != nil || u != "joao1" {
		t.Fatalf("maiúsculas devem normalizar: %q %v", u, err)
	}
}

func TestIdentificadorPareceEmail(t *testing.T) {
	if !IdentificadorPareceEmail("a@b.com") {
		t.Fatal("esperado email")
	}
	if IdentificadorPareceEmail("joao1") {
		t.Fatal("nao esperado email")
	}
}
