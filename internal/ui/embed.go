package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed all:dist
var dist embed.FS

func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "ui not embedded: "+err.Error(), http.StatusInternalServerError)
		})
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := sub.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				if looksLikeAsset(path) {
					http.NotFound(w, r)
					return
				}
				r = r.Clone(r.Context())
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

func looksLikeAsset(path string) bool {
	base := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		base = path[i+1:]
	}
	return strings.Contains(base, ".")
}
