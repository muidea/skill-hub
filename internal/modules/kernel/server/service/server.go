package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
		Handler:           localOnlyHostGuard(mux, cfg.Host),
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
