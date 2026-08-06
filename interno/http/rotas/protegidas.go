package rotas

import (
	"net/http"

	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/servicos"
)

func RegistrarProtegidas(muxPrincipal *http.ServeMux, h *handlers.Handlers, aut servicos.Autenticador, mws ...middlewares.Middleware) {
	muxProtegida := http.NewServeMux()
	muxProtegida.Handle("/v1/eu/perfil", http.HandlerFunc(h.PerfilLogado))
	muxProtegida.Handle("/v1/eu/app/presenca", http.HandlerFunc(h.PostEuAppPresenca))
	muxProtegida.Handle("/v1/eu/indique-ganhe", http.HandlerFunc(h.GetEuIndiqueGanhe))
	muxProtegida.Handle("/v1/eu/checkin-diario", http.HandlerFunc(h.CheckinDiarioEU))
	muxProtegida.Handle("/v1/eu/gire-ganhe", http.HandlerFunc(h.GireGanheEU))
	muxProtegida.Handle("/v1/eu/carteira/saldo", http.HandlerFunc(h.GetEuCarteiraSaldo))
	muxProtegida.Handle("/v1/eu/conta", http.HandlerFunc(h.ExcluirContaClienteApp))
	muxProtegida.Handle("/v1/eu/pagamentos/mercadopago/pix", http.HandlerFunc(h.PostClienteMercadoPagoPix))
	muxProtegida.Handle("/v1/eu/vouchers/calcular", http.HandlerFunc(h.PostVoucherCompraCalcular))
	muxProtegida.Handle("/v1/eu/vouchers/pagar", http.HandlerFunc(h.PostVoucherCompraPagar))
	muxProtegida.Handle("/v1/eu/vouchers/pix-retomar", http.HandlerFunc(h.GetVoucherCompraPixRetomar))
	muxProtegida.Handle("/v1/eu/vouchers/detalhe", http.HandlerFunc(h.GetVoucherCompraDetalhe))
	muxProtegida.Handle("/v1/eu/vouchers/verificar-pagamento", http.HandlerFunc(h.PostVoucherCompraVerificarPagamento))
	muxProtegida.Handle("/v1/eu/vouchers/usos-campanha", http.HandlerFunc(h.GetVoucherCompraUsosCampanha))
	muxProtegida.Handle("/v1/eu/vouchers", http.HandlerFunc(h.GetVoucherCompras))
	muxProtegida.Handle("/v1/eu/premios/resgatar", http.HandlerFunc(h.PostEuPremioResgatar))
	muxProtegida.Handle("/v1/eu/premios/resgates", http.HandlerFunc(h.GetEuPremioResgates))
	muxProtegida.Handle("/v1/eu/push/fcm", http.HandlerFunc(h.PostRegistrarTokenFCM))
	muxProtegida.Handle("/v1/eu/push/fcm/teste", http.HandlerFunc(h.PostFcmTeste))

	chain := append([]middlewares.Middleware{}, mws...)
	chain = append(chain, middlewares.ExigirAutenticacao(aut))
	muxPrincipal.Handle("/v1/eu/", middlewares.Encadear(muxProtegida, chain...))
}
