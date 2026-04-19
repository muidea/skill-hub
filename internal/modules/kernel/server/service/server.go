package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	httpapimodule "github.com/muidea/skill-hub/internal/modules/blocks/httpapi"
	webuimodule "github.com/muidea/skill-hub/internal/modules/blocks/webui"
)

type Config struct {
	Host string
	Port int
}

type Server struct {
	httpAPISvc *httpapimodule.HTTPAPI
	webUISvc   *webuimodule.WebUI
}

func New() *Server {
	return &Server{
		httpAPISvc: httpapimodule.New(),
		webUISvc:   webuimodule.New(),
	}
}

func (s *Server) Run(ctx context.Context, cfg Config) error {
	mux := http.NewServeMux()
	mux.Handle("/api/", s.httpAPISvc.Service().Handler())
	mux.Handle("/", s.webUISvc.Service().Handler())

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           secureLocalHandler(mux, cfg.Host),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func secureLocalHandler(next http.Handler, bindHost string) http.Handler {
	return securityHeaders(localOnlyBrowserGuard(localOnlyHostGuard(next, bindHost), bindHost))
}

func localOnlyHostGuard(next http.Handler, bindHost string) http.Handler {
	if !isLoopbackHost(bindHost) {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			http.Error(w, "skill-hub serve only accepts loopback host headers", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func localOnlyBrowserGuard(next http.Handler, bindHost string) http.Handler {
	if !isLoopbackHost(bindHost) {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isUnsafeMethod(r.Method) && !isAllowedBrowserWriteRequest(r) {
			http.Error(w, "skill-hub serve rejected non-loopback browser write request", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

func isAllowedBrowserWriteRequest(r *http.Request) bool {
	if site := strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")); strings.EqualFold(site, "cross-site") {
		return false
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return isLoopbackURL(origin)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return isLoopbackURL(referer)
	}
	return true
}

func isLoopbackURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	return isLoopbackHost(parsed.Host)
}

func isLoopbackHost(value string) bool {
	host := normalizeHost(value)
	if host == "" || strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func normalizeHost(value string) string {
	value = strings.TrimSpace(value)
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.Trim(value, "[]")
	value = strings.TrimSuffix(value, ".")
	return value
}
