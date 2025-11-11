package resthttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sir_venger/s3_lite/internal/config"
	"github.com/sir_venger/s3_lite/internal/repo/meta"
	"github.com/sir_venger/s3_lite/internal/usecase/filesvc"
	adapters "github.com/sir_venger/s3_lite/internal/usecase/filesvc/adapters/storage"
	"github.com/sir_venger/s3_lite/pkg/storageclient"
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
	var (
		repo filesvc.MetaStorage
		err  error
	)
	ctx := context.Background()

	metaDSN := strings.TrimSpace(cfg.MetaDSN)
	if metaDSN == "" {
		return nil, fmt.Errorf("meta_dsn is required")
	}

	repo, err = meta.NewPGStore(ctx, metaDSN)
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
