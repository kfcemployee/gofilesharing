package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kfcempoyee/gofilesharing/internal/registry/handler"
	"github.com/kfcempoyee/gofilesharing/internal/registry/repository"
	"github.com/kfcempoyee/gofilesharing/internal/registry/service"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"

	pb "github.com/kfcempoyee/gofilesharing/gen/registry/proto/v1"
)

func main() {
	// настраиваем логгер
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// настраиваем бд
	db, err := sql.Open("sqlite3", os.Getenv("DB_PATH"))
	if err != nil {
		logger.Error("failed to open db", "error", err)
		os.Exit(1)
	}

	// теперь инициализируем все слои
	repo, err := repository.NewFileRepo(db)
	if err != nil {
		logger.Error("failed to init repo", "error", err)
		os.Exit(1)
	}

	svc := service.NewFileService(repo, logger)
	h := handler.NewGRPCHandler(svc)

	// создаем контекст для всего приложения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// запускаем очистку после инициализации контекста
	svc.StartCleanup(ctx)

	// настраиваем gRpc-сервер
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterRegServiceServer(grpcServer, h)

	// запускаем сервер в горутине
	go func() {
		logger.Info("gRPC server starting on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("failed to serve", "error", err)
		}
	}()

	// начинаем слушать сигналы для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// сигнал блокирует поток тут, пока не придет сигнал
	sig := <-quit
	logger.Info("received shutdown signal", "signal", sig.String())

	// отменяем контекст - все фоновые задачи стопнуты
	cancel()
	logger.Info("background workers stopped")

	// останавливаем grpc-сервер
	logger.Info("stopping gRPC server...")
	grpcServer.GracefulStop()
	logger.Info("gRPC server stopped")

	// обрываем соединение с бд после, чтобы последнре записи успели сделаться
	if err := db.Close(); err != nil {
		logger.Error("error closing db", "error", err)
	}

	logger.Info("server stopped...")
}
