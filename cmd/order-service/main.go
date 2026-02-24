package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/order-service/app"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/cache"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/telemetry"
)

func main() {
	telemetry.InitLogger()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdown, err := telemetry.SetupTracer(ctx, getEnv("OTEL_SERVICE_NAME", "order-service"))
	if err != nil {
		slog.Error("failed to initialise tracer", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			slog.Error("tracer shutdown error", "error", err)
		}
	}()

	addr := ":" + getEnv("PORT", "9090")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed to listen", "addr", addr, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.UnaryInterceptor(interceptors.TraceServerInterceptor()),
	)

	redisAddr := getEnv("REDIS_ADDR", "redis-cache:6379")
	redisCache := cache.NewRedisCache(redisAddr, "order")
	orderSrv := app.NewOrderServer(redisCache)
	orderv1.RegisterOrderServer(grpcServer, orderSrv)

	slog.Info("order service gRPC running", "addr", addr)

	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("failed to serve", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
