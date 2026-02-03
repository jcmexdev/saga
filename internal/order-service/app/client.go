package app

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors/constants"
)

type orderServer struct {
	orderv1.UnimplementedOrderServer
	mu     sync.RWMutex
	orders map[string]*orderv1.OrderInfo
}

func NewOrderServer() *orderServer {
	return &orderServer{
		orders: make(map[string]*orderv1.OrderInfo),
	}
}

func (s *orderServer) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.NewString()

	// Calculamos el total
	var total float64
	for _, item := range req.GetItems() {
		total += float64(item.GetQuantity()) * item.GetUnitPrice()
	}

	order := &orderv1.OrderInfo{
		Id:          id,
		CustomerId:  req.GetCustomerId(),
		Items:       req.GetItems(),
		Status:      orderv1.Status_PENDING,
		TotalAmount: total,
	}

	s.orders[id] = order

	reqID := interceptors.GetMetadataValue(ctx, constants.HeaderXRequestId)
	log.Printf("Order created [request_id: %s]", reqID)
	return &orderv1.CreateOrderResponse{
		Order: order,
	}, nil
}

func (s *orderServer) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, ok := s.orders[req.GetId()]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "order %s not found", req.GetId())
	}

	return &orderv1.GetOrderResponse{
		Order: order,
	}, nil
}

func (s *orderServer) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exist := s.orders[req.GetId()]

	if !exist {
		return nil, status.Errorf(codes.NotFound, "order %s not found", req.GetId())
	}

	order.Status = req.GetStatus()

	return &orderv1.UpdateOrderStatusResponse{
		Success: true,
	}, nil
}
