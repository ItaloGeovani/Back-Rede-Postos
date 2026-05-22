package rotas

import (
	"net/http"

	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
)

func RegistrarPublicas(mux *http.ServeMux, h *handlers.Handlers, mws ...middlewares.Middleware) {
	mux.Handle("/saude", middlewares.Encadear(http.HandlerFunc(h.Saude), mws...))
	mux.Handle("/v1/autenticacao/login", middlewares.Encadear(http.HandlerFunc(h.Login), mws...))
	mux.Handle("/v1/autenticacao/login-painel", middlewares.Encadear(http.HandlerFunc(h.LoginPainelUnificadoDev), mws...))
	mux.Handle("/v1/admin-geral/dev/criar", middlewares.Encadear(http.HandlerFunc(h.CriarAdministradorGeralDev), mws...))
	mux.Handle("/v1/admin-geral/dev/login", middlewares.Encadear(http.HandlerFunc(h.LoginAdministradorGeralDev), mws...))
	mux.Handle("/v1/gestor-rede/dev/login", middlewares.Encadear(http.HandlerFunc(h.LoginGestorRedeDev), mws...))
	mux.Handle("/v1/usuario-rede/dev/login", middlewares.Encadear(http.HandlerFunc(h.LoginUsuarioRedePainelDev), mws...))
	mux.Handle("/v1/app/versao", middlewares.Encadear(http.HandlerFunc(h.VerificarVersaoAppMobile), mws...))
	mux.Handle("/v1/public/rede-cards", middlewares.Encadear(http.HandlerFunc(h.PublicListarAppCardsRede), mws...))
	mux.Handle("/v1/public/rede-info", middlewares.Encadear(http.HandlerFunc(h.PublicRedeInfo), mws...))
	mux.Handle("/v1/public/premios", middlewares.Encadear(http.HandlerFunc(h.PublicListarPremios), mws...))
	mux.Handle("/v1/public/campanhas", middlewares.Encadear(http.HandlerFunc(h.PublicListarCampanhas), mws...))
	mux.Handle("/v1/public/rede-combustiveis", middlewares.Encadear(http.HandlerFunc(h.PublicListarCombustiveisRede), mws...))
	mux.Handle("/v1/public/postos", middlewares.Encadear(http.HandlerFunc(h.PublicListarPostos), mws...))
	mux.Handle("/v1/public/clientes/cadastro", middlewares.Encadear(http.HandlerFunc(h.PublicCadastroClienteApp), mws...))
	mux.Handle("/v1/public/mercadopago/webhook/", middlewares.Encadear(http.HandlerFunc(h.MercadoPagoWebhookPublico), mws...))
	mux.Handle("/v1/public/erede/webhook/", middlewares.Encadear(http.HandlerFunc(h.ERedeWebhookPublico), mws...))
}
