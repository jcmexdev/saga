package app

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
)

// orderServer implementa orderv1.OrderInfoServiceServer
type orderServer struct {
	orderv1.UnimplementedOrderServer
	mu     sync.RWMutex
	orders map[string]*orderv1.OrderInfo
}

// NewOrderServer es el constructor
func NewOrderServer() *orderServer {
	return &orderServer{
		orders: make(map[string]*orderv1.OrderInfo),
	}
}

// CreateOrder crea una orden en memoria
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

	return &orderv1.CreateOrderResponse{
		Order: order,
	}, nil
}

// GetOrder obtiene una orden por id
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
	// verify if order exist

	order, exist := s.orders[req.GetId()]

	if !exist {
		return nil, status.Errorf(codes.NotFound, "order %s not found", req.GetId())
	}

	order.Status = req.GetStatus()

	return &orderv1.UpdateOrderStatusResponse{
		Success: true,
	}, nil
}
