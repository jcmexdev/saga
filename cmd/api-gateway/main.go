package main

import (
	"log"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/infra/adapters/service"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/infra/httpx"
	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
)

func main() {
	httpAddr := ":8080"

	orderSvcAddr := os.Getenv("ORDER_SERVICE_ADDR")
	if orderSvcAddr == "" {
		orderSvcAddr = ":9090"
	}

	// Nueva API: NewClient (sustituye a Dial/DialContext)
	conn, err := grpc.NewClient(
		orderSvcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to create gRPC client conn: %v", err)
	}
	defer conn.Close()

	grpcClient := orderv1.NewOrderServiceClient(conn)
	orderService := service.NewGRPCOrderClient(grpcClient)

	handler := httpx.NewHandler(orderService)
	router := httpx.NewRouter(handler)

	log.Println("API Gateway running on", httpAddr)
	if err := http.ListenAndServe(httpAddr, router); err != nil {
		log.Fatalf("http server failed: %v", err)
	}
}
