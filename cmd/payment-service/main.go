package main

import (
	"log"
	"net"

	paymentv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/payment/v1"
	paymentservice "github.com/jcmexdev/ecommerce-sagas/internal/payment-service/app"
	"google.golang.org/grpc"
)

func main() {
	// 1. Listener en TCP
	lis, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 2. Crear servidor gRPC
	grpcServer := grpc.NewServer()

	// 3. Crear instancia de tu implementación
	paymentClient := paymentservice.NewClient()

	// 4. Registrar el servicio en el servidor gRPC
	paymentv1.RegisterPaymentServer(grpcServer, paymentClient)

	log.Println("Payment Service gRPC running on :9091")

	// 5. Empezar a servir
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
