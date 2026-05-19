package servicos

import (
	"testing"

	"gaspass-servidor/interno/modelos"
)

func TestNormalizarPercentualArmazenamento(t *testing.T) {
	if got := normalizarPercentualArmazenamento(modelos.ModalidadeDescontoPercentual, 0.05); got != 5 {
		t.Fatalf("0.05 → 5, obteve %v", got)
	}
	if got := normalizarPercentualArmazenamento(modelos.ModalidadeDescontoPercentual, 10); got != 10 {
		t.Fatalf("10 permanece 10, obteve %v", got)
	}
	if got := normalizarPercentualArmazenamento(modelos.ModalidadeDescontoValorFixo, 0.05); got != 0.05 {
		t.Fatalf("valor fixo nao escala percentual")
	}
}
