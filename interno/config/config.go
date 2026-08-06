package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gaspass-servidor/utils"
)

type Config struct {
	Ambiente string
	// FcmCaminhoContaServico: JSON da conta de serviço do Firebase (env FCM_SA ou legado FCM_SERVICE_ACCOUNT_PATH). Vazio = não envia.
	FcmCaminhoContaServico string
	PortaHTTP              int
	// PastaPainelWeb: absoluta ou relativa com index.html do painel (PAINEL_WEB_ASSETS). Vazio = auto.
	PastaPainelWeb string
	// PublicBaseURL: URL base pública (https://api.seudominio.com) para montar webhook Mercado Pago e notification_url do PIX.
	PublicBaseURL         string
	TokenPadraoAPI        string
	AdminNomePadrao       string
	AdminEmailPadrao      string
	AdminSenhaPadrao      string
	AdminBootstrapAtivado bool
	CORSOrigemPermitida   string
	// SessaoAPIDuracao: validade de tokens tok_* (login/cadastro) quando a sessão persiste no Postgres.
	SessaoAPIDuracao time.Duration
	// APIImgBBKey: chave da API ImgBB (env API_IMGBB_KEY). Só no servidor; uploads falham se vazia.
	APIImgBBKey string
	// EvolutionGoBaseURL: URL base Evolution Go (ex. https://corenect.pro). Vazio = WhatsApp desligado.
	EvolutionGoBaseURL string
}

func Carregar() Config {
	fcm := resolverCaminhoContaFCM(
		strings.TrimSpace(utils.ObterEnv("FCM_SA", utils.ObterEnv("FCM_SERVICE_ACCOUNT_PATH", utils.ObterEnv("GOOGLE_APPLICATION_CREDENTIALS", "")))),
		strings.TrimSpace(utils.ObterEnv("FCM_BASE_DIR", "")),
	)
	return Config{
		Ambiente:               utils.ObterEnv("APP_AMBIENTE", "desenvolvimento"),
		FcmCaminhoContaServico: fcm,
		PortaHTTP:              portaHTTP(),
		PastaPainelWeb:         strings.TrimSpace(utils.ObterEnv("PAINEL_WEB_ASSETS", "")),
		TokenPadraoAPI:         utils.ObterEnv("API_TOKEN_PADRAO", "dev-super-admin"),
		AdminNomePadrao:        utils.ObterEnv("ADMIN_NOME_PADRAO", "Administrador Geral"),
		AdminEmailPadrao:       utils.ObterEnv("ADMIN_EMAIL_PADRAO", "admin@gaspass.local"),
		AdminSenhaPadrao:       utils.ObterEnv("ADMIN_SENHA_PADRAO", "123456"),
		AdminBootstrapAtivado:  utils.ObterEnv("ADMIN_BOOTSTRAP_ATIVADO", "true") == "true",
		CORSOrigemPermitida:    utils.ObterEnv("CORS_ORIGEM_PERMITIDA", "http://localhost:5173"),
		PublicBaseURL:          strings.TrimRight(strings.TrimSpace(utils.ObterEnv("PUBLIC_BASE_URL", "")), "/"),
		SessaoAPIDuracao:       duracaoSessaoAPI(),
		APIImgBBKey:            strings.TrimSpace(utils.ObterEnv("API_IMGBB_KEY", "")),
		EvolutionGoBaseURL:     strings.TrimRight(strings.TrimSpace(utils.ObterEnv("EVOLUTION_GO_BASE_URL", "")), "/"),
	}
}

// duracaoSessaoAPI: env SESSAO_DURACAO_DIAS (1–365), default 180 (6 meses).
func duracaoSessaoAPI() time.Duration {
	d := utils.ObterEnvInt("SESSAO_DURACAO_DIAS", 180)
	if d < 1 {
		d = 1
	}
	if d > 365 {
		d = 365
	}
	return time.Duration(d) * 24 * time.Hour
}

// portaHTTP: Heroku/Elastic costumam definir PORT; senao APP_PORTA; padrao 8080.
func portaHTTP() int {
	p := strings.TrimSpace(os.Getenv("PORT"))
	if p != "" {
		n, err := strconv.Atoi(p)
		if err == nil && n > 0 {
			return n
		}
	}
	return utils.ObterEnvInt("APP_PORTA", 8080)
}

// resolverCaminhoContaFCM junta caminho relativo a FCM_BASE_DIR (ex.: supervisord com CWD errado)
// e normaliza para absoluto. [raw] vazio devolve vazio.
func resolverCaminhoContaFCM(raw, baseDir string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	p := raw
	if !filepath.IsAbs(p) && baseDir != "" {
		p = filepath.Join(baseDir, p)
	}
	p = filepath.Clean(p)
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
