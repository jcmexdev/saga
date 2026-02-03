package main

import (
	"log"
	"net"

	paymentv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/payment/v1"
	paymentservice "github.com/jcmexdev/ecommerce-sagas/internal/payment-service/app"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.TraceServerInterceptor()),
	)

	paymentClient := paymentservice.NewClient()

	paymentv1.RegisterPaymentServer(grpcServer, paymentClient)

	log.Println("Payment Service gRPC running on :9091")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
