package handlers

import (
	"gaspass-servidor/interno/config"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
)

type Handlers struct {
	autenticador           servicos.Autenticador
	adminService           servicos.ServicoAdministradorGeral
	gestorService          servicos.ServicoGestorRede
	redeService            servicos.ServicoRede
	usuarioRedeService     servicos.ServicoUsuarioRede
	postoService           servicos.ServicoPosto
	campanhaService        servicos.ServicoCampanha
	premioService          servicos.ServicoPremio
	auditoriaRepo          repositorios.AuditoriaRepositorio
	estatisticasRepo       repositorios.EstatisticasPlataformaRepositorio
	appMobileRepo          repositorios.AppMobileConfigRepositorio
	appMobileRedeRepo      repositorios.AppMobileRedeRepositorio
	appCardsRepo           repositorios.AppCardsRedeRepositorio
	mpGatewayRepo          repositorios.MercadoPagoGatewayRepositorio
	voucherCompraSvc       *servicos.ServicoVoucherCompra
	combustivelRedeService *servicos.ServicoCombustivelRede
	indiqueGanhe           *servicos.ServicoIndiqueGanhe
	carteiraRepo           repositorios.CarteiraRepositorio
	niveisCliente          *servicos.ServicoNiveisCliente
	checkinDiario          *servicos.ServicoCheckinDiario
	gireGanhe              *servicos.ServicoGireGanhe
	redesSociaisRepo       repositorios.RedeLinksSociaisRepositorio
	uploadImagem           *servicos.ServicoUploadImagem
	cfg                    config.Config
}

func Novos(
	autenticador servicos.Autenticador,
	adminService servicos.ServicoAdministradorGeral,
	gestorService servicos.ServicoGestorRede,
	redeService servicos.ServicoRede,
	usuarioRedeService servicos.ServicoUsuarioRede,
	postoService servicos.ServicoPosto,
	campanhaService servicos.ServicoCampanha,
	premioService servicos.ServicoPremio,
	auditoriaRepo repositorios.AuditoriaRepositorio,
	estatisticasRepo repositorios.EstatisticasPlataformaRepositorio,
	appMobileRepo repositorios.AppMobileConfigRepositorio,
	appMobileRedeRepo repositorios.AppMobileRedeRepositorio,
	appCardsRepo repositorios.AppCardsRedeRepositorio,
	mpGatewayRepo repositorios.MercadoPagoGatewayRepositorio,
	voucherCompraSvc *servicos.ServicoVoucherCompra,
	combustivelRedeService *servicos.ServicoCombustivelRede,
	indiqueGanhe *servicos.ServicoIndiqueGanhe,
	carteiraRepo repositorios.CarteiraRepositorio,
	niveisCliente *servicos.ServicoNiveisCliente,
	checkinDiario *servicos.ServicoCheckinDiario,
	gireGanhe *servicos.ServicoGireGanhe,
	redesSociaisRepo repositorios.RedeLinksSociaisRepositorio,
	uploadImagem *servicos.ServicoUploadImagem,
	cfg config.Config,
) *Handlers {
	return &Handlers{
		autenticador:           autenticador,
		adminService:           adminService,
		gestorService:          gestorService,
		redeService:            redeService,
		usuarioRedeService:     usuarioRedeService,
		postoService:           postoService,
		campanhaService:        campanhaService,
		premioService:          premioService,
		auditoriaRepo:          auditoriaRepo,
		estatisticasRepo:       estatisticasRepo,
		appMobileRepo:          appMobileRepo,
		appMobileRedeRepo:      appMobileRedeRepo,
		appCardsRepo:           appCardsRepo,
		mpGatewayRepo:          mpGatewayRepo,
		voucherCompraSvc:       voucherCompraSvc,
		combustivelRedeService: combustivelRedeService,
		indiqueGanhe:           indiqueGanhe,
		carteiraRepo:           carteiraRepo,
		niveisCliente:          niveisCliente,
		checkinDiario:          checkinDiario,
		gireGanhe:              gireGanhe,
		redesSociaisRepo:       redesSociaisRepo,
		uploadImagem:           uploadImagem,
		cfg:                    cfg,
	}
}
