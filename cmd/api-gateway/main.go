package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/infra/adapters/service"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/infra/httpx"
	"github.com/jcmexdev/ecommerce-sagas/internal/coordinator/sagalog/sqlite"
	inventoryv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	paymentv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/payment/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/telemetry"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdown, err := telemetry.SetupTracer(ctx, getEnv("OTEL_SERVICE_NAME", "api-gateway"))
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

	dbPath := getEnv("SAGA_LOG_DB_PATH", "./data/saga.db")
	if err := os.MkdirAll("./data", 0o755); err != nil {
		slog.Error("failed to create data directory", "error", err)
		os.Exit(1)
	}

	sagaRepo, err := sqlite.Open(dbPath)
	if err != nil {
		slog.Error("failed to open saga log DB", "path", dbPath, "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := sagaRepo.Close(); err != nil {
			slog.Error("failed to close saga log DB", "error", err)
		}
	}()
	slog.Info("saga log DB ready", "path", dbPath)

	otelClientOption := grpc.WithStatsHandler(otelgrpc.NewClientHandler())

	orderConn := mustDial(getEnv("ORDER_SERVICE_ADDR", ":9090"), otelClientOption)
	defer orderConn.Close()

	payConn := mustDial(getEnv("PAYMENT_SERVICE_ADDR", ":9091"), otelClientOption)
	defer payConn.Close()

	invConn := mustDial(getEnv("INVENTORY_SERVICE_ADDR", ":9092"), otelClientOption)
	defer invConn.Close()

	orderClient := orderv1.NewOrderClient(orderConn)
	payClient := paymentv1.NewPaymentClient(payConn)
	invClient := inventoryv1.NewInventoryClient(invConn)

	orderService := service.NewGRPCOrderClient(orderClient)

	handler := httpx.NewHandler(orderService, orderClient, payClient, invClient, sagaRepo)
	router := httpx.NewRouter(handler)

	httpAddr := getEnv("HTTP_ADDR", ":8080")
	slog.Info("API Gateway (Orchestrator) running", "addr", httpAddr)

	if err := http.ListenAndServe(httpAddr, router); err != nil {
		slog.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// mustDial creates a gRPC client connection or exits the process on failure.
func mustDial(addr string, extraOpts ...grpc.DialOption) *grpc.ClientConn {
	opts := append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, extraOpts...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		slog.Error("could not connect to gRPC service", "addr", addr, "error", err)
		os.Exit(1)
	}
	return conn
}
