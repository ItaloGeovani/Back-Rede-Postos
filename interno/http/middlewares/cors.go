package middlewares

import (
	"net/http"
	"strings"
)

func parseOrigensPermitidas(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{"http://localhost:5173"}
	}
	partes := strings.Split(raw, ",")
	out := make([]string, 0, len(partes))
	for _, p := range partes {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"http://localhost:5173"}
	}
	return out
}

func origemPermitidaParaRequest(origemReq string, permitidas []string) string {
	origemReq = strings.TrimSpace(origemReq)
	if origemReq == "" {
		// Sem Origin (ex.: same-origin / curl): ecoa a primeira permitida.
		return permitidas[0]
	}
	for _, p := range permitidas {
		if strings.EqualFold(p, origemReq) {
			return origemReq
		}
	}
	return ""
}

// CORS ecoa um único Access-Control-Allow-Origin (exigência do browser).
// CORS_ORIGEM_PERMITIDA aceita lista separada por vírgula.
func CORS(origemPermitida string) Middleware {
	permitidas := parseOrigensPermitidas(origemPermitida)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origem := origemPermitidaParaRequest(r.Header.Get("Origin"), permitidas)
			if origem != "" {
				w.Header().Set("Access-Control-Allow-Origin", origem)
			}
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-Painel-Web")
			w.Header().Set("Access-Control-Max-Age", "600")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
