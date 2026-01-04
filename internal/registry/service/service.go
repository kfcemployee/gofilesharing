package service

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kfcempoyee/gofilesharing/internal/registry/domain"
)

// интерфейс репо имеет 4 метода - сохранить, отдать, удалить файл и очистить хранилище
type FileRepoInterface interface {
	Insert(ctx context.Context, file *domain.File) error
	Get(ctx context.Context, shortName string) (*domain.File, error)
	Delete(ctx context.Context, id string) error
	ClearExpired(ctx context.Context) error
}

// сервис должен содержать экземпляр репо и логгер (можно сделать новый или прокинуть общий)
type FileService struct {
	Repo   FileRepoInterface
	Logger *slog.Logger
}

// передаем в сервис репо и логгер
func NewFileService(repo FileRepoInterface, logger *slog.Logger) *FileService {
	return &FileService{
		Repo:   repo,
		Logger: logger,
	}
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// генерирует короткий айди для ссылки
// быстрее через builder
func generateId(length int) string {
	var b strings.Builder
	b.Grow(length)

	for range length {
		b.WriteByte(charset[rand.Intn(len(charset))])
	}

	return b.String()
}

// загрузить файл в бд
func (s *FileService) Upload(ctx context.Context, uuid, name string, size int64, contentType string) (string, error) {
	fileId := generateId(5) // генерируем айди

	storagePath := filepath.Join("data/storage", uuid+".dat")
	err := os.MkdirAll(storagePath, 0755)

	if err != nil {
		return "", err
	}

	err = os.Rename(filepath.Join("data/tmp", uuid), storagePath)

	if err != nil {
		s.Logger.Error("error uploading a file", "error", err)
		return "", domain.ErrInService
	}

	newFile := domain.File{
		ID:           fileId,
		OriginalName: name,
		StoragePath:  storagePath,
		Size:         size,
		ContentType:  contentType,
		CreatedAt:    time.Now(),
	}

	err = s.Repo.Insert(ctx, &newFile)

	if err != nil {
		s.Logger.Error("error uploading a file", "error", err)
		return "", domain.ErrInRepo
	}

	s.Logger.Info("uploaded file: " + uuid)
	return fileId, nil
}

// берем путь и данные файла в бд по короткому имени
func (s *FileService) Get(ctx context.Context, id string) (*domain.File, error) {
	resFile, err := s.Repo.Get(ctx, id)

	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}

		s.Logger.Error("error getting a file", "error", err)
		return nil, domain.ErrInRepo
	}

	s.Logger.Info("got a file: success")
	return resFile, nil
}

// запуск очищения диска и бд от просроченных записей
func (s *FileService) StartCleanup(ctx context.Context) {
	s.Logger.Info("starting cleanup worker")

	go func() {
		s.Logger.Info("starting initial cleanup")
		if err := s.Repo.ClearExpired(ctx); err != nil {
			s.Logger.Error("initial cleanup failed", "", err)
		}

		ti := time.NewTicker(24 * time.Hour) // каждые 24 часа система очищает от истекших файлов
		defer ti.Stop()

		for {
			select {
			case <-ti.C:
				s.Logger.Info("starting scheduled cleanup")

				if err := s.Repo.ClearExpired(ctx); err != nil {
					s.Logger.Error("scheduled cleanup failed", "", err)
				}
			case <-ctx.Done():
				s.Logger.Info("cleanup stopped due to cancelled context", "", ctx.Err())
				return
			}
		}
	}()
}
