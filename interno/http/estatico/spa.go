package estatico

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PossiveisPastasPainel percorridas em ordem; usa a primeira em que existir index.html (cwd do processo).
var PossiveisPastasPainel = []string{
	"assets",
	"./assets",
	"painel-web-dist",
	"./painel-web-dist",
	"../painel-web-react/dist",
	"./../painel-web-react/dist",
	"dist",
	"./dist",
}

// EncontrarRaizPainel retorna o diretorio absoluto com index.html do build Vite, ou string vazia.
// override: env PAINEL_WEB_ASSETS (caminho absoluto ou relativo ao cwd do processo).
func EncontrarRaizPainel(override string) string {
	override = strings.TrimSpace(override)
	if override != "" {
		idx := filepath.Join(override, "index.html")
		if fi, err := os.Stat(idx); err == nil && !fi.IsDir() {
			if abs, err := filepath.Abs(override); err == nil {
				return abs
			}
			return override
		}
	}
	for _, dir := range PossiveisPastasPainel {
		idx := filepath.Join(dir, "index.html")
		fi, err := os.Stat(idx)
		if err != nil || fi.IsDir() {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return dir
		}
		return abs
	}
	return ""
}

const htmlPainelAusente = `<!DOCTYPE html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>GasPass — painel nao implantado</title>
<style>body{font-family:system-ui,sans-serif;max-width:42rem;margin:2rem auto;padding:0 1rem;line-height:1.5;color:#1e293b}
code{background:#f1f5f9;padding:.15rem .35rem;border-radius:4px;font-size:.9em}
</style></head><body>
<h1>API no ar — painel web ausente</h1>
<p>O servidor nao encontrou a pasta do build do React (<code>index.html</code>). Sem ela, a raiz do site retorna esta pagina em vez do painel.</p>
<p><strong>O que fazer:</strong> na maquina de deploy, rode no projeto <code>painel-web-react</code>:</p>
<pre style="background:#f8fafc;padding:1rem;border-radius:8px;overflow:auto">npm ci
npm run build:deploy</pre>
<p>Isso copia o Vite <code>dist/</code> para <code>servidor-go/assets/</code>. Envie essa pasta no deploy (ou remova <code>assets/</code> do <code>.gitignore</code> e faca commit do build).</p>
<p>Opcional no servidor: variavel <code>PAINEL_WEB_ASSETS</code> apontando para o diretorio que contem <code>index.html</code>.</p>
<p>API: <a href="/saude">/saude</a></p>
</body></html>`

// ComSPAFallback encerra GET/HEAD nao-API com arquivos estaticos e index.html (rotas do React).
// Rotas /saude, /v1/..., /releases/... e /indique|/landing/... seguem sempre para o mux da API.
func ComSPAFallback(api http.Handler, staticRoot string) http.Handler {
	bypass := func(path string) bool {
		return path == "/saude" ||
			strings.HasPrefix(path, "/v1/") ||
			strings.HasPrefix(path, "/releases/") ||
			path == "/indique" ||
			strings.HasPrefix(path, "/indique/") ||
			strings.HasPrefix(path, "/landing/")
	}
	if staticRoot == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if bypass(path) {
				api.ServeHTTP(w, r)
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				api.ServeHTTP(w, r)
				return
			}
			if path == "/" || !strings.Contains(path, ".") {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, htmlPainelAusente)
				return
			}
			api.ServeHTTP(w, r)
		})
	}
	staticRoot = filepath.Clean(staticRoot)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if bypass(path) {
			api.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			api.ServeHTTP(w, r)
			return
		}
		if path == "/" {
			http.ServeFile(w, r, filepath.Join(staticRoot, "index.html"))
			return
		}
		rel := strings.TrimPrefix(path, "/")
		if rel == "" || strings.Contains(rel, "..") {
			http.Error(w, "caminho invalido", http.StatusBadRequest)
			return
		}
		full := filepath.Join(staticRoot, filepath.FromSlash(rel))
		fullClean := filepath.Clean(full)
		rootClean := filepath.Clean(staticRoot)
		relPath, err := filepath.Rel(rootClean, fullClean)
		if err != nil || strings.HasPrefix(relPath, "..") {
			http.Error(w, "acesso negado", http.StatusForbidden)
			return
		}
		if fi, err := os.Stat(fullClean); err == nil && !fi.IsDir() {
			http.ServeFile(w, r, fullClean)
			return
		}
		if strings.Contains(path, ".") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(staticRoot, "index.html"))
	})
}

// ResolverDirReleases retorna o diretorio absoluto de artefactos do updater, ou string vazia.
func ResolverDirReleases(override string) string {
	candidatos := []string{}
	if o := strings.TrimSpace(override); o != "" {
		candidatos = append(candidatos, o)
	}
	candidatos = append(candidatos,
		"releases",
		"./releases",
		filepath.Join("..", "releases"),
	)
	for _, dir := range candidatos {
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return dir
		}
		return abs
	}
	return ""
}

// ResolverDirLanding retorna o diretorio absoluto da pasta landing/ (HTML estático Indique e ganhe).
func ResolverDirLanding() string {
	candidatos := []string{
		"landing",
		"./landing",
		filepath.Join("..", "landing"),
	}
	for _, dir := range candidatos {
		idx := filepath.Join(dir, "indique", "index.html")
		fi, err := os.Stat(idx)
		if err != nil || fi.IsDir() {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return dir
		}
		return abs
	}
	return ""
}
