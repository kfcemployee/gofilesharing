package gateway

import (
	"log/slog"
	"net/http"
)

type FileProvider interface {
	GetFile(w http.ResponseWriter, r *http.Request)
	UploadFile(w http.ResponseWriter, r *http.Request)
	GetInfo(w http.ResponseWriter, r *http.Request)
}

type FileRouter struct {
	h FileProvider
}

func NewRouter(handler FileProvider) *FileRouter {
	return &FileRouter{h: handler}
}

func (r *FileRouter) Route(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/get/{id}/", r.h.GetFile)
	mux.HandleFunc("/get/{id}/info/", r.h.GetInfo)
	mux.HandleFunc("/upload", r.h.UploadFile)

	return enableCORS(loggingMiddleware(logger, mux))
}
