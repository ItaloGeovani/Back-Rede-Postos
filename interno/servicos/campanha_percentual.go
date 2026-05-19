package servicos

import "gaspass-servidor/interno/modelos"

// normalizarPercentualArmazenamento grava percentuais na escala 0–100.
// Aceita legado em fração (ex.: 0.05 → 5) apenas na modalidade PERCENTUAL.
func normalizarPercentualArmazenamento(modalidade string, valor float64) float64 {
	if modalidade != modelos.ModalidadeDescontoPercentual {
		return valor
	}
	if valor > 0 && valor <= 1 {
		return valor * 100
	}
	return valor
}
