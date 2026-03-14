package service

import (
	"context"
	"fmt"
	"net/http"
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
		Handler:           mux,
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
