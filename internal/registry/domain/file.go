package domain

import "time"

// основная структура файла
type File struct {
	ID           string
	OriginalName string
	StoragePath  string
	Size         int64
	ContentType  string
	CreatedAt    time.Time
}
