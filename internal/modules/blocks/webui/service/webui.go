package service

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets/*
var assets embed.FS

type WebUI struct{}

func New() *WebUI {
	return &WebUI{}
}

func (w *WebUI) Handler() http.Handler {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(sub))
}
