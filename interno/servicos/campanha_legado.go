package servicos

import (
	"regexp"
	"strings"

	"gaspass-servidor/interno/modelos"
)

var reHTMLTags = regexp.MustCompile(`<[^>]*>`)

// aplicarLegadoCampanhaCashback corrige campanhas ainda gravadas como DESCONTO com texto de cashback.
// Preferir migração 047 no banco; isto cobre ambientes onde a migração ainda não rodou.
func aplicarLegadoCampanhaCashback(c *modelos.Campanha) {
	if c == nil {
		return
	}
	if strings.TrimSpace(c.TipoBeneficio) == modelos.TipoBeneficioCashback {
		return
	}
	if !pareceCampanhaCashbackLegado(c) {
		return
	}
	c.TipoBeneficio = modelos.TipoBeneficioCashback
}

func pareceCampanhaCashbackLegado(c *modelos.Campanha) bool {
	if strings.ToUpper(strings.TrimSpace(c.ModalidadeDesconto)) != modelos.ModalidadeDescontoPercentual {
		return false
	}
	base := strings.ToUpper(strings.TrimSpace(c.BaseDesconto))
	if base != "" && base != modelos.BaseDescontoValorCompra {
		return false
	}
	texto := strings.ToLower(strings.Join([]string{
		c.Nome,
		c.Titulo,
		c.TituloExibicao,
		reHTMLTags.ReplaceAllString(c.Descricao, " "),
	}, " "))
	return strings.Contains(texto, "cashback") || strings.Contains(texto, "cash back")
}
