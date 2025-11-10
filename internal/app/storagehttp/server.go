package storagehttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Server serves the storage node HTTP API on top of the local filesystem.
type Server struct {
	dataDir string
}

// New создаёт HTTP-обработчик стоража поверх каталога с данными.
func New(dataDir string) http.Handler {
	srv := &Server{
		dataDir: dataDir,
	}

	return srv.routes()
}

// routes регистрирует обработчики для частей, здоровья и GC.
func (a *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Route("/parts/{fileID}/{idx}", func(pr chi.Router) {
		// Accept both PUT and POST to stay compatible with the documented API and older clients.
		pr.Put("/", a.insertPart)
		pr.Get("/", a.fetchPart)
		pr.Head("/", a.inspectPart)
	})

	r.Get("/health", a.health)
	r.HandleFunc("/admin/gc", a.gcOnce)

	return r
}
