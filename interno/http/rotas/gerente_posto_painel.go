package rotas

import (
	"net/http"

	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
)

// RegistrarGerentePostoPainel mesmas capacidades do painel do gestor, exceto cadastro de rede/postos/gestores e moeda virtual.
func RegistrarGerentePostoPainel(muxPrincipal *http.ServeMux, h *handlers.Handlers, aut servicos.Autenticador, mws ...middlewares.Middleware) {
	mux := http.NewServeMux()
	mux.Handle("/v1/gerente-posto/dev/rede", http.HandlerFunc(h.MinhaRedeGestorRede))
	mux.Handle("/v1/gerente-posto/dev/campanhas/listar", http.HandlerFunc(h.ListarCampanhasGestorRede))
	mux.Handle("/v1/gerente-posto/dev/campanhas/criar", http.HandlerFunc(h.CriarCampanhaGestorRede))
	mux.Handle("/v1/gerente-posto/dev/campanhas/editar", http.HandlerFunc(h.EditarCampanhaGestorRede))
	mux.Handle("/v1/gerente-posto/dev/postos/listar", http.HandlerFunc(h.ListarPostosGestorRede))
	mux.Handle("/v1/gerente-posto/dev/postos/gateway-meios", http.HandlerFunc(h.EditarMeiosPostoGestorRede))
	mux.Handle("/v1/gerente-posto/dev/premios/listar", http.HandlerFunc(h.ListarPremiosGestorRede))
	mux.Handle("/v1/gerente-posto/dev/premios/criar", http.HandlerFunc(h.CriarPremioGestorRede))
	mux.Handle("/v1/gerente-posto/dev/premios/editar", http.HandlerFunc(h.EditarPremioGestorRede))
	mux.Handle("/v1/gerente-posto/dev/premios/resgates/listar", http.HandlerFunc(h.GetPremioResgatesPainel))
	mux.Handle("/v1/gerente-posto/dev/premios/resgates/entregar", http.HandlerFunc(h.PostPremioResgateEntregar))
	mux.Handle("/v1/gerente-posto/dev/premios/resgates/cancelar", http.HandlerFunc(h.PostPremioResgateCancelar))
	mux.Handle("/v1/gerente-posto/dev/clientes/presenca-app", http.HandlerFunc(h.ListarClientesPresencaAppPainel))
	mux.Handle("/v1/gerente-posto/dev/clientes/carteira", http.HandlerFunc(h.ListarClientesCarteiraPainel))
	mux.Handle("/v1/gerente-posto/dev/usuarios-rede/listar", http.HandlerFunc(h.ListarUsuariosRedeGestor))
	mux.Handle("/v1/gerente-posto/dev/usuarios-rede/criar-equipe", http.HandlerFunc(h.CriarUsuarioEquipeGestorRede))
	mux.Handle("/v1/gerente-posto/dev/usuarios-rede/editar-equipe", http.HandlerFunc(h.EditarUsuarioEquipeGestorRede))
	mux.Handle("/v1/gerente-posto/dev/relatorios/resumo", http.HandlerFunc(h.ResumoRelatoriosGestorRede))
	mux.Handle("/v1/gerente-posto/dev/auditoria/listar", http.HandlerFunc(h.ListarAuditoriaGestorRede))
	mux.Handle("/v1/gerente-posto/dev/eventos-operacionais/listar", http.HandlerFunc(h.ListarEventosOperacionaisGestor))
	mux.Handle("/v1/gerente-posto/dev/app-cards", http.HandlerFunc(h.AppCardsGestorRede))
	mux.Handle("/v1/gerente-posto/dev/combustiveis/listar", http.HandlerFunc(h.ListarCombustiveisRede))
	mux.Handle("/v1/gerente-posto/dev/combustiveis/criar", http.HandlerFunc(h.CriarCombustivelRede))
	mux.Handle("/v1/gerente-posto/dev/combustiveis/editar", http.HandlerFunc(h.EditarCombustivelRede))
	mux.Handle("/v1/gerente-posto/dev/combustiveis/excluir", http.HandlerFunc(h.ExcluirCombustivelRede))
	mux.Handle("/v1/gerente-posto/dev/push/fcm/rede/teste", http.HandlerFunc(h.PostFcmTesteRedePainel))
	mux.Handle("/v1/gerente-posto/dev/push/diagnostico", http.HandlerFunc(h.GetPushDiagnosticoRedePainel))
	mux.Handle("/v1/gerente-posto/dev/vouchers/consultar", http.HandlerFunc(h.GetVoucherConsultaPorCodigoEquipe))
	mux.Handle("/v1/gerente-posto/dev/vouchers/baixar", http.HandlerFunc(h.PostVoucherBaixaEquipe))
	mux.Handle("/v1/gerente-posto/dev/vouchers/listar", http.HandlerFunc(h.GetVouchersComprasPainelEquipe))
	mux.Handle("/v1/gerente-posto/dev/mercadopago-gateway", http.HandlerFunc(h.MercadoPagoGatewayGestor))
	mux.Handle("/v1/gerente-posto/dev/mercadopago-gateway/posto", http.HandlerFunc(h.MercadoPagoGatewayPostoGestor))
	mux.Handle("/v1/gerente-posto/dev/erede-gateway", http.HandlerFunc(h.ERedeGatewayGestor))
	mux.Handle("/v1/gerente-posto/dev/erede-gateway/posto", http.HandlerFunc(h.ERedeGatewayPostoGestor))

	chain := append([]middlewares.Middleware{}, mws...)
	chain = append(chain, middlewares.ExigirAutenticacao(aut), middlewares.ExigirPapel(modelos.PapelGerentePosto))
	muxPrincipal.Handle("/v1/gerente-posto/", middlewares.Encadear(mux, chain...))
}
