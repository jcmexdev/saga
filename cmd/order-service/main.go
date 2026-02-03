package main

import (
	"log"
	"net"

	"google.golang.org/grpc"

	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/order-service/app"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors"
)

func main() {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.TraceServerInterceptor()),
	)

	orderSrv := app.NewOrderServer()

	orderv1.RegisterOrderServer(grpcServer, orderSrv)

	log.Println("OrderService gRPC running on :9090")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
