package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/kfcempoyee/gofilesharing/internal/registry/domain"

	_ "github.com/mattn/go-sqlite3"
)

// setupDB создает чистое in-memory соединение для каждого теста
func setupDB(t *testing.T) (*FileRepo, *sql.DB, func()) {
	// Используем :memory: для быстрого тестирования без создания файлов БД на диске
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}

	repo, err := NewFileRepo(db)
	if err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	// Функция очистки (teardown)
	cleanup := func() {
		db.Close()
	}

	return repo, db, cleanup
}

func TestNewFileRepo(t *testing.T) {
	repo, db, cleanup := setupDB(t)
	defer cleanup()

	if repo == nil {
		t.Fatal("Repository is nil")
	}

	// Проверяем, создалась ли таблица
	// sqlite_master - системная таблица SQLite
	row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='files';")
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("Table 'files' was not created: %v", err)
	}
}

func TestFileRepo_InsertAndGet(t *testing.T) {
	repo, _, cleanup := setupDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC() // SQLite хранит время в UTC по умолчанию

	file := &domain.File{
		ID:           "id-123",
		OriginalName: "test.txt",
		StoragePath:  "/tmp/test.txt",
		Size:         1024,
		ContentType:  "text/plain",
		CreatedAt:    now,
	}

	// 1. Тест Insert
	if err := repo.Insert(ctx, file); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// 2. Тест Get (успешный)
	fetched, err := repo.Get(ctx, file.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Сравниваем поля
	if fetched.ID != file.ID {
		t.Errorf("Expected ID %s, got %s", file.ID, fetched.ID)
	}
	if fetched.OriginalName != file.OriginalName {
		t.Errorf("Expected OriginalName %s, got %s", file.OriginalName, fetched.OriginalName)
	}

	// Проверяем время (учитывая, что БД может отсечь наносекунды)
	if fetched.CreatedAt.Unix() != file.CreatedAt.Unix() {
		t.Errorf("CreatedAt mismatch")
	}
}

func TestFileRepo_Get_NotFound(t *testing.T) {
	repo, _, cleanup := setupDB(t)
	defer cleanup()

	_, err := repo.Get(context.Background(), "non-existent-id")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestFileRepo_Delete(t *testing.T) {
	repo, _, cleanup := setupDB(t)
	defer cleanup()

	ctx := context.Background()
	file := &domain.File{
		ID:           "del-123",
		OriginalName: "del.txt",
		StoragePath:  "path",
		CreatedAt:    time.Now(),
	}

	// Вставляем
	if err := repo.Insert(ctx, file); err != nil {
		t.Fatal(err)
	}

	// Удаляем
	if err := repo.Delete(ctx, file.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Проверяем, что файла больше нет
	_, err := repo.Get(ctx, file.ID)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound after delete, got %v", err)
	}
}
