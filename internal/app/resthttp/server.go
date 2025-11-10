package resthttp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/yourname/storage_lite/internal/config"
	meta "github.com/yourname/storage_lite/internal/repo"
	"github.com/yourname/storage_lite/internal/usecase/filesvc"
	adapters "github.com/yourname/storage_lite/internal/usecase/filesvc/adapters/storage"
	"github.com/yourname/storage_lite/pkg/storageclient"
)

const defaultFileParts = 6

type Server struct {
	FilesService filesvc.Service
	Cfg          *config.Config
}

type addStoragesRequest struct {
	Storages []string `json:"storages"`
}

// NewServer конструктор
func NewServer(cfg *config.Config) (http.Handler, *Server, error) {
	files, err := buildFileService(cfg)
	if err != nil {
		return nil, nil, err
	}

	srv := &Server{
		FilesService: files,
		Cfg:          cfg,
	}

	rtr := chi.NewRouter()
	rtr.Post("/files", srv.postFiles)
	rtr.Get("/files/{id}", srv.getFile)
	rtr.Get("/admin/config", func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(cfg) })
	rtr.Post("/admin/storages", srv.addStorages)

	return rtr, srv, nil
}

func buildFileService(cfg *config.Config) (filesvc.Service, error) {
	repo, err := meta.Open(cfg.MetaPath)
	if err != nil {
		return nil, err
	}

	cli := storageclient.New()
	adapter := adapters.NewHealthAdapter(0)
	r := filesvc.NewRouter(adapter)

	fileManager := filesvc.New(filesvc.Deps{
		MetaStorage: repo,
		Router:      r,
		StorageCli:  cli,
		Parts:       defaultFileParts,
	})

	fileManager.Router.Set(cfg.Storages)
	return fileManager, nil
}

func (s *Server) addStorages(w http.ResponseWriter, r *http.Request) {
	var payload addStoragesRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(payload.Storages) == 0 {
		http.Error(w, "storages list is empty", http.StatusBadRequest)
		return
	}

	s.FilesService.AddStorages(payload.Storages...)
	w.WriteHeader(http.StatusNoContent)
}
