package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/kfcempoyee/gofilesharing/internal/registry/domain"
)

// репозиторий содержит указатель на бд и реализует интерфейс FileRepoInterface
type FileRepo struct {
	db *sql.DB
}

const (
	tableName = "files" // имя таблицы для удобства
)

// инициализация (создание таблиц) происходит прямо при создании репозитория
func NewFileRepo(db *sql.DB) (*FileRepo, error) {
	query := `CREATE TABLE IF NOT EXISTS files (
	id TEXT PRIMARY KEY,        
	original_name TEXT NOT NULL, 
	storage_path TEXT NOT NULL,  
	size_bytes INTEGER,           
	content_type TEXT,           
	created_at DATETIME,   
	expired_at TIMESTAMP
);`
	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to set up table: %w", err)
	}

	return &FileRepo{
		db: db,
	}, nil
}

// сохранить файл, вернуть nil в случае удачи, error в противном случае
func (f *FileRepo) Insert(ctx context.Context, file *domain.File) error {
	query := "INSERT INTO " + tableName + " (id, original_name, storage_path, size_bytes, content_type, created_at, expired_at)" +
		"VALUES (?, ?, ?, ?, ?, ?, ?);"

	_, err := f.db.ExecContext(
		ctx, query,
		file.ID,
		file.OriginalName,
		file.StoragePath,
		file.Size,
		file.ContentType,
		file.CreatedAt,
		file.CreatedAt.Add(48*time.Hour), // файл живёт 48 часов, потом удаляется
	)

	return err
}

// взять файл или ошибку
func (f *FileRepo) Get(ctx context.Context, shortName string) (*domain.File, error) {
	query := "SELECT * FROM " + tableName + " WHERE id = ?;"
	respFile := domain.File{}

	var exp time.Time
	err := f.db.QueryRowContext(ctx, query, shortName).Scan(
		&respFile.ID,
		&respFile.OriginalName,
		&respFile.StoragePath,
		&respFile.Size,
		&respFile.ContentType,
		&respFile.CreatedAt,
		&exp,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(exp) {
		return nil, domain.ErrExpired
	}

	return &respFile, nil
}

// удалить файл из бд, вернуть nil, если получилось, в противном случае ошибку (несуществующий айди ошибкой не является).
func (f *FileRepo) Delete(ctx context.Context, id string) error {

	query := "DELETE FROM " + tableName + " WHERE id = ?"
	_, err := f.db.ExecContext(ctx, query, id)
	return err
}

// почистить истекшие файлы
func (f *FileRepo) ClearExpired(ctx context.Context) error {
	query := "SELECT storage_path FROM " + tableName + " WHERE expired_at < DATETIME('now', 'localtime');"
	rows, err := f.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return err
		}

		// если возникла ошибка, просто пропускаем операцию, не прерывая цикл
		_ = os.Remove(path)
	}

	if rows.Err() != nil {
		return rows.Err()
	}

	query = "DELETE FROM " + tableName + " WHERE expired_at < DATETIME('now', 'localtime');"
	_, err = f.db.ExecContext(ctx, query)
	return err
}
