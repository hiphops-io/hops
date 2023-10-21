package httpserver

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiphops-io/hops/assets"
	"github.com/rs/zerolog"
)

type consoleController struct {
	Logger     zerolog.Logger
	PathPrefix string
}

// The console router serves the single page app for the console.
// It will serve the index.html file for any path that does not exist,
// allowing client-side to handle routing
func ConsoleRouter(logger zerolog.Logger) chi.Router {
	r := chi.NewRouter()

	controller := &consoleController{
		Logger:     logger,
		PathPrefix: "/console",
	}

	r.HandleFunc("/*", controller.handle())
	return r
}

func (c *consoleController) loadContentDir() http.FileSystem {
	content, err := fs.Sub(assets.Console, "console")
	if err != nil {
		c.Logger.Fatal().Msg("Unable to load console UI")
	}
	return http.FS(content)
}

func (c *consoleController) handle() func(http.ResponseWriter, *http.Request) {
	content := c.loadContentDir()
	fs := http.FileServer(content)
	statichandler := http.StripPrefix(c.PathPrefix, fs)

	return func(w http.ResponseWriter, r *http.Request) {
		wt := &intercept404{ResponseWriter: w}
		statichandler.ServeHTTP(wt, r)

		if wt.statusCode == http.StatusNotFound {
			r.URL.Path = c.PathPrefix
			w.Header().Set("Content-Type", "text/html")
			statichandler.ServeHTTP(w, r)
		}
	}
}

type intercept404 struct {
	http.ResponseWriter
	statusCode int
}

func (w *intercept404) Write(b []byte) (int, error) {
	if w.statusCode == http.StatusNotFound {
		return len(b), nil
	}
	if w.statusCode != 0 {
		w.WriteHeader(w.statusCode)
	}
	return w.ResponseWriter.Write(b)
}

func (w *intercept404) WriteHeader(statusCode int) {
	if statusCode >= 300 && statusCode < 400 {
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}
	w.statusCode = statusCode
}
