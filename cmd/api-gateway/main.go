package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/infra/adapters/service"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/infra/httpx"
	inventoryv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	paymentv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	httpAddr := ":8080"

	orderAddr := getEnv("ORDER_SERVICE_ADDR", ":9090")
	paymentAddr := getEnv("PAYMENT_SERVICE_ADDR", ":9091")
	inventoryAddr := getEnv("INVENTORY_SERVICE_ADDR", ":9092")

	orderConn := createGRPCConn(orderAddr)
	defer orderConn.Close()

	payConn := createGRPCConn(paymentAddr)
	defer payConn.Close()

	invConn := createGRPCConn(inventoryAddr)
	defer invConn.Close()

	orderClient := orderv1.NewOrderClient(orderConn)
	payClient := paymentv1.NewPaymentClient(payConn)
	invClient := inventoryv1.NewInventoryClient(invConn)

	orderService := service.NewGRPCOrderClient(orderClient)

	handler := httpx.NewHandler(orderService, orderClient, payClient, invClient)
	router := httpx.NewRouter(handler)

	log.Printf("API Gateway (Orchestrator) running on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, router); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func createGRPCConn(addr string) *grpc.ClientConn {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to %s: %v", addr, err)
	}
	return conn
}
