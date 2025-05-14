package main

import (
	"log/slog"
	"os"

	"github.com/ianschenck/envflag"
	pbEvent "github.com/rx3lixir/gateway-service/gateway-grpc/gen/go/event"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var (
		//db_url_env    = envflag.String("DB_URL", "postgres://", "!")
		grpc_svc_addr = envflag.String("GRPC_ADDR", "0.0.0.0:9091", "!")
	)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(*grpc_svc_addr, opts...)
	if err != nil {
		slog.Error("failed to connect to server", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	pbEvent.NewEventServiceClient(conn)

}
