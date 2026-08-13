package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var reUsuarioCliente = regexp.MustCompile(`^[a-z0-9]{4,}$`)

// NormalizarUsuarioCliente deixa só a-z0-9 em minúsculas (remove o resto).
func NormalizarUsuarioCliente(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ValidarUsuarioCliente exige exatamente a-z0-9 (min. 4), permitindo só diferença de maiúsculas na entrada.
// Devolve o usuário normalizado (minúsculas) para gravação.
func ValidarUsuarioCliente(s string) (string, error) {
	raw := strings.ToLower(strings.TrimSpace(s))
	u := NormalizarUsuarioCliente(s)
	if u != raw || !reUsuarioCliente.MatchString(u) {
		return "", fmt.Errorf("usuario deve ter no minimo 4 caracteres, apenas letras minusculas e numeros (sem acento, espaco ou simbolos)")
	}
	return u, nil
}

// IdentificadorPareceEmail detecta login legado por e-mail.
func IdentificadorPareceEmail(s string) bool {
	return strings.Contains(strings.TrimSpace(s), "@")
}
