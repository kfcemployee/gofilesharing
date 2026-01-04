package domain

import "errors"

// отдельный файл с ошибками, вынесено в отдельный файл на сервисном слое. в хендлер поступают
// ошибки именно отсюда. если надо будет добавить обрабатывать больше ошибок - просто
// описать новую в этом файле.

var (
	ErrNotFound  = errors.New("file not found") // файл не найден
	ErrExpired   = errors.New("link expired")   // сслыка недействительна
	ErrInRepo    = errors.New("repo error")
	ErrInService = errors.New("error in service")
)
