package filesvc

import (
	"context"
	"io"

	"github.com/sir_venger/s3_lite/internal/models"
	"github.com/sir_venger/s3_lite/pkg/storageclient"
)

type (
	// MetaStorage хранилище мета данных файлов
	MetaStorage interface {
		Get(ctx context.Context, id string) (models.File, error)
		Save(ctx context.Context, file models.File) error
	}

	// Service объединяет операции по загрузке и выдаче файлов.
	Service interface {
		UploadWhole(ctx context.Context, r io.Reader, size int64, name string) (models.UploadResult, error)
		Stream(ctx context.Context, fileID string, w io.Writer) error
		AddStorages(storages ...string)
	}
)

// Deps описывает внешние компоненты, которые требуются сервису файлов.
type Deps struct {
	MetaStorage MetaStorage
	Router      *Router
	StorageCli  storageclient.Client
	Parts       int
}

// Files реализует бизнес-логику сервиса загрузки на базе переданных зависимостей.
type Files struct {
	Deps
}

// New конструирует сервис загрузки с заданными зависимостями.
func New(deps Deps) *Files {
	return &Files{Deps: deps}
}

var _ Service = (*Files)(nil)

// AddStorages добавляет новые стораджи в маршрутизатор без удаления существующих.
func (s *Files) AddStorages(storages ...string) {
	if s.Router == nil || len(storages) == 0 {
		return
	}
	s.Router.Add(storages...)
}
