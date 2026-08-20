package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gaspass-servidor/interno/config"
	"gaspass-servidor/interno/http/estatico"
	"gaspass-servidor/interno/http/handlers"
	"gaspass-servidor/interno/http/middlewares"
	"gaspass-servidor/interno/http/rotas"
	"gaspass-servidor/interno/notificacoes"
	"gaspass-servidor/interno/repositorios"
	"gaspass-servidor/interno/servicos"
	"gaspass-servidor/utils"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Aplicacao struct {
	cfg      config.Config
	servidor *http.Server
	banco    *sql.DB
}

// verificarCaminhoContaFCM confirma no arranque que o JSON existe e e ficheiro (log no terminal).
// Se FCM_SA (ou FCM_SERVICE_ACCOUNT_PATH) estiver vazio, nao e erro; push fica desativado.
// Tenta varios candidatos: caminho do .env, CWD, pasta do executavel (hospedagem com CWD diferente da app).
// Em sucesso, actualiza [cfg].FcmCaminhoContaServico para o caminho absoluto que abriu.
func verificarCaminhoContaFCM(cfg *config.Config) error {
	p := strings.TrimSpace(cfg.FcmCaminhoContaServico)
	if p == "" {
		log.Print("fcm: FCM_SA nao definido; envio de push desativado")
		return nil
	}
	cands := candidatosCaminhoFcm(p)
	for _, c := range cands {
		info, err := os.Stat(c)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		abs, errA := filepath.Abs(c)
		if errA != nil {
			abs = c
		}
		cfg.FcmCaminhoContaServico = abs
		log.Printf("fcm: credenciais OK: %q (%d bytes)", abs, info.Size())
		return nil
	}
	// Padrao: a API sobe sem push. Com FCM_EXIGE_FICHEIRO=1, falha se o ficheiro nao existir (CI/producao rigorosa).
	if utils.ObterEnvSimNao("FCM_EXIGE_FICHEIRO", false) {
		linhas := make([]string, 0, len(cands))
		for i, c := range cands {
			if i >= 20 {
				linhas = append(linhas, fmt.Sprintf("... (+%d)", len(cands)-i))
				break
			}
			linhas = append(linhas, c)
		}
		b := filepath.Base(p)
		return fmt.Errorf("fcm: ficheiro nao encontrado em %d caminhos. Procurou-se %q. Amostra: %s. (Carregue o .json no servidor, ou nao defina FCM_EXIGE_FICHEIRO.)", len(cands), b, strings.Join(linhas, " | "))
	}
	primeiro := ""
	if len(cands) > 0 {
		primeiro = cands[0]
	}
	log.Printf("fcm: AVISO: ficheiro nao encontrado; push FCM desativado. Carregue o JSON da conta de servico no servidor e reinicie, ou ajuste FCM_SA. Tentou %d caminhos (ex.: %q)", len(cands), primeiro)
	cfg.FcmCaminhoContaServico = ""
	return nil
}

func candidatosCaminhoFcm(p string) []string {
	p = strings.TrimSpace(p)
	if p == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if a, e := filepath.Abs(s); e == nil {
			s = a
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	nome := filepath.Base(p)
	add(p)
	if !filepath.IsAbs(p) {
		if w, e := os.Getwd(); e == nil {
			add(filepath.Join(w, p))
		}
	}
	// CWD: subir diretorios com o nome do ficheiro (json noutro nivel que o processo)
	if w, e := os.Getwd(); e == nil {
		candidatoSubirPastas(filepath.Clean(w), nome, add)
	}
	// executavel: a partir do bin real (nunca subir a partir do cache "go run", evita dezenas de caminhos inuteis)
	if ex, e := os.Executable(); e == nil {
		ex = filepath.Clean(ex)
		if !strings.Contains(ex, "go-build") {
			dirEx := filepath.Dir(ex)
			if !filepath.IsAbs(p) {
				add(filepath.Join(dirEx, p))
			}
			candidatoSubirPastas(dirEx, nome, add)
		}
		// se "go run", so CWD + FCM_BASE_DIR + p definem os candidatos (json na pasta do projecto)
	}
	return out
}

// candidatoSubirPastas adiciona dir/nome, parent/nome, ... (ate 7 passos) para achar o json noutro nivel da arvore.
func candidatoSubirPastas(dir0, nome string, add func(string)) {
	d := dir0
	for i := 0; i < 7; i++ {
		if d == "" {
			return
		}
		add(filepath.Join(d, nome))
		parent := filepath.Dir(d)
		if parent == d {
			return
		}
		d = parent
	}
}

func Nova() (*Aplicacao, error) {
	cfg := config.Carregar()
	if err := verificarCaminhoContaFCM(&cfg); err != nil {
		return nil, err
	}
	banco, err := abrirBanco()
	if err != nil {
		return nil, err
	}

	autenticador := servicos.NovoAutenticadorTokenComPersistencia(banco, cfg.TokenPadraoAPI, cfg.SessaoAPIDuracao)
	repoAdmin := repositorios.NovoAdministradorGeralPostgres(banco)
	repoGestor := repositorios.NovoGestorRedePostgres(banco)
	repoRede := repositorios.NovoRedePostgres(banco)
	repoUsuarioRede := repositorios.NovoUsuarioRedePostgres(banco)
	repoPosto := repositorios.NovoPostoPostgres(banco)
	repoCampanha := repositorios.NovoCampanhaPostgres(banco)
	repoPremio := repositorios.NovoPremioPostgres(banco)
	repoAuditoria := repositorios.NovoAuditoriaPostgres(banco)
	estatisticasPlataforma := repositorios.NovoEstatisticasPlataformaPostgres(banco)
	repoAppMobile := repositorios.NovoAppMobileConfigPostgres(banco)
	repoAppMobileRede := repositorios.NovoAppMobileRedePostgres(banco)
	repoAppCards := repositorios.NovoAppCardsRedePostgres(banco)
	repoMercadoPagoGateway := repositorios.NovoMercadoPagoGatewayPostgres(banco)
	repoERedeGateway := repositorios.NovoERedeGatewayPostgres(banco)
	repoVoucherCompra := repositorios.NovoVoucherCompraPostgres(banco)
	repoCombustivelRede := repositorios.NovoCombustivelRedePostgres(banco)
	repoCarteira := repositorios.NovoCarteiraPostgres(banco)
	repoIndiqueGanhe := repositorios.NovoIndiqueGanhePostgres(banco)
	repoNiveisCliente := repositorios.NovoNiveisClientePostgres(banco)
	repoGireGanhe := repositorios.NovoGireGanhePostgres(banco)
	repoLinksSociais := repositorios.NovoRedeLinksSociaisPostgres(banco)
	svcNiveisCliente := servicos.NovoServicoNiveisCliente(repoNiveisCliente)
	svcIndiqueGanhe := servicos.NovoServicoIndiqueGanhe(repoRede, repoIndiqueGanhe, repoCarteira, repoUsuarioRede, svcNiveisCliente)
	repoCheckinDiario := repositorios.NovoCheckinDiarioPostgres(banco)
	svcAdmin, err := servicos.NovoServicoAdministradorGeral(repoAdmin, autenticador)
	if err != nil {
		banco.Close()
		return nil, err
	}
	svcRede := servicos.NovoServicoRede(repoRede)
	svcGestor, err := servicos.NovoServicoGestorRede(repoGestor, repoRede, autenticador)
	if err != nil {
		banco.Close()
		return nil, err
	}
	svcUsuarioRede, err := servicos.NovoServicoUsuarioRede(repoUsuarioRede, repoRede, autenticador, svcIndiqueGanhe)
	if err != nil {
		banco.Close()
		return nil, err
	}
	svcCheckinDiario := servicos.NovoServicoCheckinDiario(banco, repoCheckinDiario, repoCarteira, svcNiveisCliente, svcUsuarioRede)
	svcGireGanhe := servicos.NovoServicoGireGanhe(banco, repoGireGanhe, repoCarteira, svcNiveisCliente, svcUsuarioRede)
	svcPosto := servicos.NovoServicoPosto(repoPosto, repoRede)
	svcCampanha := servicos.NovoServicoCampanha(repoCampanha, repoRede, repoCombustivelRede)
	svcVoucherCompra := servicos.NovoServicoVoucherCompra(repoVoucherCompra, svcCampanha, repoMercadoPagoGateway, repoERedeGateway, repoRede, repoPosto, repoCarteira, repoCombustivelRede, repoUsuarioRede, cfg, svcIndiqueGanhe)
	svcCombustivelRede := servicos.NovoServicoCombustivelRede(repoCombustivelRede, repoRede)
	svcPremio := servicos.NovoServicoPremio(repoPremio, repoRede)
	repoPremioResgate := repositorios.NovoPremioResgatePostgres(banco)
	svcPremioResgate := servicos.NovoServicoPremioResgate(banco, repoPremio, repoPremioResgate, repoCarteira, repoRede)
	repoEventosOp := repositorios.NovoEventosOperacionaisPostgres(banco)
	repoWhatsApp := repositorios.NovoWhatsAppNotificacoesPostgres(banco)
	waNotifier := &notificacoes.WhatsAppNotifier{
		BaseURL: cfg.EvolutionGoBaseURL,
		CfgRepo: repoWhatsApp,
	}
	svcEventosOp := servicos.NovoServicoEventosOperacionais(banco, repoEventosOp, repoWhatsApp, waNotifier, repoRede, repoPosto)
	svcVoucherCompra.DefinirEventosOperacionais(svcEventosOp)
	if err := bootstrapAdminPadrao(cfg, svcAdmin); err != nil {
		banco.Close()
		return nil, err
	}

	svcUploadImagem := servicos.NovoServicoUploadImagem(cfg)
	h := handlers.Novos(autenticador, svcAdmin, svcGestor, svcRede, svcUsuarioRede, svcPosto, svcCampanha, svcPremio, repoAuditoria, estatisticasPlataforma, repoAppMobile, repoAppMobileRede, repoAppCards, repoMercadoPagoGateway, repoERedeGateway, svcVoucherCompra, svcCombustivelRede, svcIndiqueGanhe, repoCarteira, svcNiveisCliente, svcCheckinDiario, svcGireGanhe, svcPremioResgate, repoLinksSociais, svcUploadImagem, svcEventosOp, cfg)

	muxPrincipal := http.NewServeMux()
	mwGlobal := []middlewares.Middleware{
		middlewares.CORS(cfg.CORSOrigemPermitida),
		middlewares.RequestID(),
		middlewares.RecuperacaoPanico(),
		middlewares.Logger(),
	}

	rotas.RegistrarPublicas(muxPrincipal, h, mwGlobal...)
	rotas.RegistrarProtegidas(muxPrincipal, h, autenticador, mwGlobal...)
	rotas.RegistrarPrivadas(muxPrincipal, h, autenticador, mwGlobal...)
	rotas.RegistrarGestorRedePainel(muxPrincipal, h, autenticador, mwGlobal...)
	rotas.RegistrarGerentePostoPainel(muxPrincipal, h, autenticador, mwGlobal...)
	rotas.RegistrarFrentistaPainel(muxPrincipal, h, autenticador, mwGlobal...)

	releasesDir := estatico.ResolverDirReleases(cfg.PastaReleases)
	if releasesDir != "" {
		log.Printf("releases estaticos: %s (GET /releases/)", releasesDir)
		fs := http.StripPrefix("/releases/", http.FileServer(http.Dir(releasesDir)))
		muxPrincipal.Handle("/releases/", middlewares.Encadear(fs, mwGlobal...))
	} else {
		log.Printf("releases: pasta nao encontrada (RELEASES_DIR ou ./releases)")
	}

	landingDir := estatico.ResolverDirLanding()
	if landingDir != "" {
		log.Printf("landing indique: %s (GET /indique , /landing/)", landingDir)
		indiqueIndex := filepath.Join(landingDir, "indique", "index.html")
		muxPrincipal.Handle("/indique", middlewares.Encadear(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.Error(w, "metodo nao permitido", http.StatusMethodNotAllowed)
				return
			}
			http.ServeFile(w, r, indiqueIndex)
		}), mwGlobal...))
		muxPrincipal.Handle("/indique/", middlewares.Encadear(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.Error(w, "metodo nao permitido", http.StatusMethodNotAllowed)
				return
			}
			http.ServeFile(w, r, indiqueIndex)
		}), mwGlobal...))
		landingFS := http.StripPrefix("/landing/", http.FileServer(http.Dir(landingDir)))
		muxPrincipal.Handle("/landing/", middlewares.Encadear(landingFS, mwGlobal...))
	} else {
		log.Printf("landing: pasta nao encontrada (./landing/indique/index.html)")
	}

	raizPainel := estatico.EncontrarRaizPainel(cfg.PastaPainelWeb)
	if raizPainel != "" {
		log.Printf("painel web estatico: %s (GET /)", raizPainel)
	} else {
		log.Printf("painel web: nenhuma pasta com index.html (assets/, PAINEL_WEB_ASSETS ou build:deploy); raiz mostra aviso HTML")
	}
	handlerPrincipal := estatico.ComSPAFallback(muxPrincipal, raizPainel)

	servidor := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.PortaHTTP),
		Handler:           handlerPrincipal,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Aplicacao{
		cfg:      cfg,
		servidor: servidor,
		banco:    banco,
	}, nil
}

// testr
func bootstrapAdminPadrao(cfg config.Config, svc servicos.ServicoAdministradorGeral) error {
	if !cfg.AdminBootstrapAtivado {
		return nil
	}

	_, err := svc.Criar(cfg.AdminNomePadrao, cfg.AdminEmailPadrao, cfg.AdminSenhaPadrao)
	if err == nil {
		log.Printf("admin geral padrao criado: email=%s", cfg.AdminEmailPadrao)
		return nil
	}
	if errors.Is(err, repositorios.ErrEmailJaCadastrado) {
		return nil
	}
	return err
}

func (a *Aplicacao) Executar(ctx context.Context) error {
	errCh := make(chan error, 1)

	log.Printf("servidor rodando em http://localhost%s (ambiente=%s)", a.servidor.Addr, a.cfg.Ambiente)

	go func() {
		if err := a.servidor.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.servidor.Shutdown(ctxShutdown); err != nil {
			return err
		}
		return a.banco.Close()
	case err := <-errCh:
		return err
	}
}

func abrirBanco() (*sql.DB, error) {
	dsn, err := utils.MontarDSNPostgres()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
