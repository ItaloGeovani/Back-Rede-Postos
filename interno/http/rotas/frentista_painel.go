package rotas

import (
	"net/http"

	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
)

// RegistrarFrentistaPainel leitura da rede, campanhas, postos (contexto) e relatorios — sem criacao ou edicao.
func RegistrarFrentistaPainel(muxPrincipal *http.ServeMux, h *handlers.Handlers, aut servicos.Autenticador, mws ...middlewares.Middleware) {
	mux := http.NewServeMux()
	mux.Handle("/v1/frentista/dev/rede", http.HandlerFunc(h.MinhaRedeGestorRede))
	mux.Handle("/v1/frentista/dev/campanhas/listar", http.HandlerFunc(h.ListarCampanhasGestorRede))
	mux.Handle("/v1/frentista/dev/postos/listar", http.HandlerFunc(h.ListarPostosGestorRede))
	mux.Handle("/v1/frentista/dev/relatorios/resumo", http.HandlerFunc(h.ResumoRelatoriosGestorRede))
	mux.Handle("/v1/frentista/dev/vouchers/consultar", http.HandlerFunc(h.GetVoucherConsultaPorCodigoEquipe))
	mux.Handle("/v1/frentista/dev/vouchers/listar", http.HandlerFunc(h.GetVouchersComprasPainelEquipe))

	chain := append([]middlewares.Middleware{}, mws...)
	chain = append(chain, middlewares.ExigirAutenticacao(aut), middlewares.ExigirPapel(modelos.PapelFrentista))
	muxPrincipal.Handle("/v1/frentista/", middlewares.Encadear(mux, chain...))
}
