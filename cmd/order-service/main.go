package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/order-service/app"
)

func main() {
	// 1. Listener en TCP
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 2. Crear servidor gRPC
	grpcServer := grpc.NewServer()

	// 3. Crear instancia de tu implementación
	orderSrv := app.NewOrderServer()

	// 4. Registrar el servicio en el servidor gRPC
	orderv1.RegisterOrderServiceServer(grpcServer, orderSrv)

	log.Println("OrderService gRPC running on :9090")

	// 5. Empezar a servir
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
