package rotas

import (
	"net/http"

	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
)

func RegistrarGestorRedePainel(muxPrincipal *http.ServeMux, h *handlers.Handlers, aut servicos.Autenticador, mws ...middlewares.Middleware) {
	mux := http.NewServeMux()
	mux.Handle("/v1/gestor-rede/dev/rede", http.HandlerFunc(h.MinhaRedeGestorRede))
	mux.Handle("/v1/gestor-rede/dev/gestores", http.HandlerFunc(h.ListarGestoresDaMinhaRede))
	mux.Handle("/v1/gestor-rede/dev/campanhas/listar", http.HandlerFunc(h.ListarCampanhasGestorRede))
	mux.Handle("/v1/gestor-rede/dev/campanhas/criar", http.HandlerFunc(h.CriarCampanhaGestorRede))
	mux.Handle("/v1/gestor-rede/dev/campanhas/editar", http.HandlerFunc(h.EditarCampanhaGestorRede))
	mux.Handle("/v1/gestor-rede/dev/postos/listar", http.HandlerFunc(h.ListarPostosGestorRede))
	mux.Handle("/v1/gestor-rede/dev/postos/criar", http.HandlerFunc(h.CriarPostoGestorRede))
	mux.Handle("/v1/gestor-rede/dev/premios/listar", http.HandlerFunc(h.ListarPremiosGestorRede))
	mux.Handle("/v1/gestor-rede/dev/premios/criar", http.HandlerFunc(h.CriarPremioGestorRede))
	mux.Handle("/v1/gestor-rede/dev/premios/editar", http.HandlerFunc(h.EditarPremioGestorRede))
	mux.Handle("/v1/gestor-rede/dev/redes/moeda-virtual", http.HandlerFunc(h.EditarMoedaVirtualMinhaRedeGestor))
	mux.Handle("/v1/gestor-rede/dev/redes/config-voucher", http.HandlerFunc(h.EditarVoucherConfigMinhaRedeGestor))
	mux.Handle("/v1/gestor-rede/dev/redes/app-modulos", http.HandlerFunc(h.EditarAppModulosMinhaRedeGestor))
	mux.Handle("/v1/gestor-rede/dev/redes/indique-ganhe", http.HandlerFunc(h.IndiqueGanheConfigGestor))
	mux.Handle("/v1/gestor-rede/dev/redes/checkin-diario", http.HandlerFunc(h.CheckinDiarioConfigGestor))
	mux.Handle("/v1/gestor-rede/dev/redes/gire-ganhe", http.HandlerFunc(h.GireGanheConfigGestor))
	mux.Handle("/v1/gestor-rede/dev/redes/redes-sociais", http.HandlerFunc(h.RedesSociaisGestor))
	mux.Handle("/v1/gestor-rede/dev/redes/niveis-cliente", http.HandlerFunc(h.NiveisClienteConfigGestor))
	mux.Handle("/v1/gestor-rede/dev/usuarios-rede/listar", http.HandlerFunc(h.ListarUsuariosRedeGestor))
	mux.Handle("/v1/gestor-rede/dev/usuarios-rede/criar-equipe", http.HandlerFunc(h.CriarUsuarioEquipeGestorRede))
	mux.Handle("/v1/gestor-rede/dev/usuarios-rede/editar-equipe", http.HandlerFunc(h.EditarUsuarioEquipeGestorRede))
	mux.Handle("/v1/gestor-rede/dev/relatorios/resumo", http.HandlerFunc(h.ResumoRelatoriosGestorRede))
	mux.Handle("/v1/gestor-rede/dev/auditoria/listar", http.HandlerFunc(h.ListarAuditoriaGestorRede))
	mux.Handle("/v1/gestor-rede/dev/app-cards", http.HandlerFunc(h.AppCardsGestorRede))
	mux.Handle("/v1/gestor-rede/dev/mercadopago-gateway", http.HandlerFunc(h.MercadoPagoGatewayGestor))
	mux.Handle("/v1/gestor-rede/dev/combustiveis/listar", http.HandlerFunc(h.ListarCombustiveisRede))
	mux.Handle("/v1/gestor-rede/dev/combustiveis/criar", http.HandlerFunc(h.CriarCombustivelRede))
	mux.Handle("/v1/gestor-rede/dev/combustiveis/editar", http.HandlerFunc(h.EditarCombustivelRede))
	mux.Handle("/v1/gestor-rede/dev/combustiveis/excluir", http.HandlerFunc(h.ExcluirCombustivelRede))
	mux.Handle("/v1/gestor-rede/dev/push/fcm/rede/teste", http.HandlerFunc(h.PostFcmTesteRedePainel))
	mux.Handle("/v1/gestor-rede/dev/push/diagnostico", http.HandlerFunc(h.GetPushDiagnosticoRedePainel))
	mux.Handle("/v1/gestor-rede/dev/vouchers/consultar", http.HandlerFunc(h.GetVoucherConsultaPorCodigoEquipe))
	mux.Handle("/v1/gestor-rede/dev/vouchers/baixar", http.HandlerFunc(h.PostVoucherBaixaEquipe))
	mux.Handle("/v1/gestor-rede/dev/vouchers/listar", http.HandlerFunc(h.GetVouchersComprasPainelEquipe))

	chain := append([]middlewares.Middleware{}, mws...)
	chain = append(chain, middlewares.ExigirAutenticacao(aut), middlewares.ExigirPapel(modelos.PapelGestorRede))
	muxPrincipal.Handle("/v1/gestor-rede/", middlewares.Encadear(mux, chain...))
}
