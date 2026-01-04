package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	pb "github.com/kfcempoyee/gofilesharing/gen/registry/proto/v1"
	"github.com/kfcempoyee/gofilesharing/internal/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	lg := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(lg)

	conn, err := grpc.NewClient("localhost:5051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewRegServiceClient(conn)

	handler := &gateway.FileHandler{
		TmpDir:     "./data/tmp",
		GRpcClient: client,
		Logger:     lg,
	}

	router := gateway.NewRouter(handler)
	mux := router.Route(lg)

	if err = http.ListenAndServe(":8080", mux); err != nil {
		lg.Error("error with gateway")
		return
	}
}
