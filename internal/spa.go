package internal

import (
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Single Page Applications usually return the contents from "index.html"
// for paths not matching a file resource directly.
func spaMiddleware(indexFile string) func(handler http.Handler) http.Handler {
	index, err := os.OpenFile(filepath.Clean(indexFile), os.O_RDONLY, 0400)
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			wrw := new(wrappedRW)
			wrw.ResponseWriter = w
			next.ServeHTTP(wrw, r)
			if wrw.nf {
				if err != nil {
					// no index file available; redirect to root
					http.Redirect(w, r, "/", http.StatusFound)
					return
				}
				// serve index file
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.ServeContent(w, r, index.Name(), time.Now(), index)
			}
		}
		return http.HandlerFunc(fn)
	}
}

type wrappedRW struct {
	http.ResponseWriter
	nf bool
}

func (wrw *wrappedRW) WriteHeader(status int) {
	if status == http.StatusNotFound {
		wrw.nf = true // don't write the 404 header
	} else {
		wrw.ResponseWriter.WriteHeader(status)
	}
}

func (wrw *wrappedRW) Write(p []byte) (int, error) {
	if wrw.nf {
		return len(p), nil // no-op
	}
	return wrw.ResponseWriter.Write(p)
}
