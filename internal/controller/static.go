package controller

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) WithStatic(handler http.Handler) http.Handler {
	if s.cfg.WebDir == "" {
		return handler
	}
	static := spaFileServer(s.cfg.WebDir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" {
			handler.ServeHTTP(w, r)
			return
		}
		static.ServeHTTP(w, r)
	})
}

func spaFileServer(root string) http.Handler {
	fileServer := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if path == "." {
			path = "index.html"
		}
		fullPath := filepath.Join(root, path)
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
