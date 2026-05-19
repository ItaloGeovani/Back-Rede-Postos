package servicos

import (
	"testing"

	"gaspass-servidor/interno/modelos"
)

func TestAplicarLegadoCampanhaCashback(t *testing.T) {
	c := &modelos.Campanha{
		Nome:               "Compre com CashBack",
		Titulo:             "Compre com CashBack",
		TipoBeneficio:      modelos.TipoBeneficioDesconto,
		ModalidadeDesconto: modelos.ModalidadeDescontoPercentual,
		BaseDesconto:       modelos.BaseDescontoValorCompra,
		ValorDesconto:      0.1,
	}
	aplicarLegadoCampanhaCashback(c)
	if c.TipoBeneficio != modelos.TipoBeneficioCashback {
		t.Fatalf("esperava CASHBACK, obteve %q", c.TipoBeneficio)
	}

	c2 := &modelos.Campanha{
		Nome:               "Desconto de 5%",
		TipoBeneficio:      modelos.TipoBeneficioDesconto,
		ModalidadeDesconto: modelos.ModalidadeDescontoPercentual,
		BaseDesconto:       modelos.BaseDescontoValorCompra,
	}
	aplicarLegadoCampanhaCashback(c2)
	if c2.TipoBeneficio != modelos.TipoBeneficioDesconto {
		t.Fatalf("desconto comum nao deve virar cashback")
	}
}
