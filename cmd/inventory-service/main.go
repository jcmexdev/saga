package main

import (
	"log"
	"net"

	inventoryv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
	inventoryservice "github.com/jcmexdev/ecommerce-sagas/internal/inventory-service"
	"google.golang.org/grpc"
)

func main() {
	// 1. Listener en TCP
	lis, err := net.Listen("tcp", ":9092")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 2. Crear servidor gRPC
	grpcServer := grpc.NewServer()

	// 3. Crear instancia de tu implementación
	paymentClient := inventoryservice.NewClient()

	// 4. Registrar el servicio en el servidor gRPC
	inventoryv1.RegisterInventoryServer(grpcServer, paymentClient)

	log.Println("Inventory Service gRPC running on :9092")

	// 5. Empezar a servir
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
