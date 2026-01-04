package handler

import (
	"context"
	"errors"

	pb "github.com/kfcempoyee/gofilesharing/gen/registry/proto/v1"
	"github.com/kfcempoyee/gofilesharing/internal/registry/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// интерфейс сервиса
type FileServiceInterface interface {
	Upload(ctx context.Context, uuid string, name string, size int64, contentType string) (string, error)
	Get(ctx context.Context, id string) (*domain.File, error)
	StartCleanup(ctx context.Context)
}

// сам хендлер должен принять сервис и структуру для совместимости
type GrpcHandler struct {
	service FileServiceInterface
	pb.UnimplementedRegServiceServer
}

// при создании хендлера укажем сервис
func NewGRPCHandler(s FileServiceInterface) *GrpcHandler {
	return &GrpcHandler{
		service: s,
	}
}

// взять путь к файлу в памяти по его короткому айди
func (h *GrpcHandler) GetFile(ctx context.Context, req *pb.GetFileDataReq) (*pb.GetFileDataResp, error) {
	file, err := h.service.Get(ctx, req.GetShortName())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "File not found.")
		}

		if errors.Is(err, domain.ErrExpired) {
			return nil, status.Error(codes.DeadlineExceeded, "Link expired")
		}

		return nil, status.Error(codes.Internal, "Internal Error.")
	}

	return &pb.GetFileDataResp{
		StPath:      file.StoragePath,
		Filename:    file.OriginalName,
		SizeBytes:   file.Size,
		ContentType: file.ContentType,
	}, nil
}

// сохранить файл из временного пути в память и записать в бд
func (h *GrpcHandler) RegisterFile(ctx context.Context, req *pb.RegisterFileRequest) (*pb.RegisterFileResp, error) {
	sn, err := h.service.Upload(
		ctx,
		req.GetTmpName(),
		req.GetFilename(),
		req.GetSizeBytes(),
		req.GetContentType(),
	)

	if err != nil {
		return nil, status.Error(codes.Internal, "Internal Error")
	}

	return &pb.RegisterFileResp{ShortName: sn}, nil
}
