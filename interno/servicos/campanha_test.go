package servicos

import (
	"testing"

	"gaspass-servidor/interno/modelos"
)

func TestValidarDescontoEUso_CashbackExigePercentual(t *testing.T) {
	ok := validarDescontoEUso(
		true, false,
		modelos.TipoBeneficioCashback,
		modelos.ModalidadeDescontoValorFixo,
		modelos.BaseDescontoValorCompra,
		2,
		10,
		nil,
	)
	if ok {
		t.Fatalf("cashback com modalidade fixa deveria ser invalido")
	}

	ok = validarDescontoEUso(
		true, false,
		modelos.TipoBeneficioCashback,
		modelos.ModalidadeDescontoPercentual,
		modelos.BaseDescontoValorCompra,
		0.5,
		10,
		nil,
	)
	if !ok {
		t.Fatalf("cashback com modalidade percentual deveria ser valido")
	}
}
