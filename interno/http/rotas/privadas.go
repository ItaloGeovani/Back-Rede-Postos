package rotas

import (
	"net/http"

	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/modelos"
	"gaspass-servidor/interno/servicos"
)

func RegistrarPrivadas(muxPrincipal *http.ServeMux, h *handlers.Handlers, aut servicos.Autenticador, mws ...middlewares.Middleware) {
	muxPrivada := http.NewServeMux()
	muxPrivada.Handle("/v1/admin/diagnostico", http.HandlerFunc(h.DiagnosticoAdmin))
	muxPrivada.Handle("/v1/admin/dashboard/resumo", http.HandlerFunc(h.ResumoDashboardAdmin))
	muxPrivada.Handle("/v1/admin/relatorios/resumo", http.HandlerFunc(h.ResumoRelatoriosPlataformaAdmin))
	muxPrivada.Handle("/v1/admin/auditoria/listar", http.HandlerFunc(h.ListarAuditoriaPlataformaAdmin))
	muxPrivada.Handle("/v1/admin/sistema/configuracao", http.HandlerFunc(h.ConfiguracaoSistemaAdmin))
	muxPrivada.Handle("/v1/admin/app-mobile/versao", http.HandlerFunc(h.AppMobileVersaoAdmin))
	muxPrivada.Handle("/v1/admin/administradores-gerais/dev/editar", http.HandlerFunc(h.EditarAdministradorGeralDev))
	muxPrivada.Handle("/v1/admin/gestores-rede/dev/listar", http.HandlerFunc(h.ListarGestoresRedeDev))
	muxPrivada.Handle("/v1/admin/gestores-rede/dev/criar", http.HandlerFunc(h.CriarGestorRedeComPlanoDev))
	muxPrivada.Handle("/v1/admin/gestores-rede/dev/editar", http.HandlerFunc(h.EditarGestorRedeComPlanoDev))
	muxPrivada.Handle("/v1/admin/redes/dev/listar", http.HandlerFunc(h.ListarRedesDev))
	muxPrivada.Handle("/v1/admin/redes/dev/criar", http.HandlerFunc(h.CriarRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/editar", http.HandlerFunc(h.EditarRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/ativar", http.HandlerFunc(h.AtivarRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/desativar", http.HandlerFunc(h.DesativarRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/moeda-virtual", http.HandlerFunc(h.EditarMoedaVirtualRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/config-voucher", http.HandlerFunc(h.EditarVoucherConfigRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/app-modulos", http.HandlerFunc(h.EditarAppModulosRedeDev))
	muxPrivada.Handle("/v1/admin/redes/dev/app-versao", http.HandlerFunc(h.AppMobileVersaoRedeAdmin))
	muxPrivada.Handle("/v1/admin/redes/dev/vouchers/listar", http.HandlerFunc(h.GetVouchersComprasPainelAdmin))
	muxPrivada.Handle("/v1/admin/redes/dev/push/diagnostico", http.HandlerFunc(h.GetPushDiagnosticoRedeAdmin))
	muxPrivada.Handle("/v1/admin/usuarios-rede/dev/listar", http.HandlerFunc(h.ListarUsuariosRedeDev))
	muxPrivada.Handle("/v1/admin/usuarios-rede/dev/criar-equipe", http.HandlerFunc(h.CriarUsuarioEquipeRedeDev))
	muxPrivada.Handle("/v1/admin/usuarios-rede/dev/editar-equipe", http.HandlerFunc(h.EditarUsuarioEquipeRedeDev))
	muxPrivada.Handle("/v1/admin/postos/dev/listar", http.HandlerFunc(h.ListarPostosRedeDev))
	muxPrivada.Handle("/v1/admin/postos/dev/criar", http.HandlerFunc(h.CriarPostoRedeDev))
	muxPrivada.Handle("/v1/admin/postos/dev/editar", http.HandlerFunc(h.EditarPostoRedeDev))
	muxPrivada.Handle("/v1/admin/campanhas/dev/listar", http.HandlerFunc(h.ListarCampanhasRedeDev))
	muxPrivada.Handle("/v1/admin/campanhas/dev/criar", http.HandlerFunc(h.CriarCampanhaRedeDev))
	muxPrivada.Handle("/v1/admin/campanhas/dev/editar", http.HandlerFunc(h.EditarCampanhaRedeDev))
	muxPrivada.Handle("/v1/admin/premios/dev/listar", http.HandlerFunc(h.ListarPremiosRedeDev))
	muxPrivada.Handle("/v1/admin/premios/dev/criar", http.HandlerFunc(h.CriarPremioRedeDev))
	muxPrivada.Handle("/v1/admin/premios/dev/editar", http.HandlerFunc(h.EditarPremioRedeDev))
	muxPrivada.Handle("/v1/admin/upload-imagem", http.HandlerFunc(h.PostUploadImagem))

	chain := append([]middlewares.Middleware{}, mws...)
	chain = append(chain, middlewares.ExigirAutenticacao(aut), middlewares.ExigirPapel(modelos.PapelSuperAdmin))
	muxPrincipal.Handle("/v1/admin/", middlewares.Encadear(muxPrivada, chain...))
}
