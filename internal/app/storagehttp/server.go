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
		// Эндпоинты Storage API для загрузки, чтения и инспекции конкретной части.
		pr.Put("/", a.insertPart)
		pr.Get("/", a.fetchPart)
		pr.Head("/", a.inspectPart)
	})

	// health нужен REST-сервису, чтобы проверять состояние стораджа.
	r.Get("/health", a.health)
	// /admin/gc позволяет вручную очистить зависшие директории.
	r.HandleFunc("/admin/gc", a.gcOnce)

	return r
}
