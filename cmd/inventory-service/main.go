package main

import (
	"log"
	"net"

	inventoryv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
	inventoryservice "github.com/jcmexdev/ecommerce-sagas/internal/inventory-service"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":9092")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.TraceServerInterceptor()),
	)

	paymentClient := inventoryservice.NewClient()

	inventoryv1.RegisterInventoryServer(grpcServer, paymentClient)

	log.Println("Inventory Service gRPC running on :9092")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
