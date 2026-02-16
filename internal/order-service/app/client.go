package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/order-service/adapters/grpc/mappers"
	"github.com/jcmexdev/ecommerce-sagas/internal/order-service/domain"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/cache"
)

// orderServer is the gRPC server implementation for the Order service.
// It uses an in-memory map as storage for demonstration purposes.
type orderServer struct {
	orderv1.UnimplementedOrderServer
	mu     sync.RWMutex
	orders map[string]*domain.Order
	cache  cache.Cache
}

// Ensure orderServer implements the gRPC interface at compile time.
var _ orderv1.OrderServer = (*orderServer)(nil)

// NewOrderServer creates a new in-memory order gRPC server.
func NewOrderServer(cacheProvider cache.Cache) *orderServer {
	return &orderServer{
		orders: make(map[string]*domain.Order),
		cache:  cacheProvider,
	}
}

func (s *orderServer) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	newOrder := mappers.OrderFromProto(ctx, req)

	// Idempotency check via cache (fast path).
	cacheKey := s.cache.GenerateKey("create", newOrder.IdempotencyKey)
	if newOrder.IdempotencyKey != "" {
		cacheValue, _ := s.cache.Get(ctx, cacheKey)
		if cacheValue != "" {
			var cached domain.Order
			if err := json.Unmarshal([]byte(cacheValue), &cached); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to decode cached order: %v", err)
			}
			return &orderv1.CreateOrderResponse{Order: mappers.OrderToProto(&cached)}, nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Idempotency check via in-memory map (slow path, handles cache misses).
	if newOrder.IdempotencyKey != "" {
		for _, existing := range s.orders {
			if existing.IdempotencyKey == newOrder.IdempotencyKey {
				return &orderv1.CreateOrderResponse{Order: mappers.OrderToProto(existing)}, nil
			}
		}
	}

	s.orders[newOrder.ID] = newOrder

	// Populate cache asynchronously to avoid blocking the response.
	if newOrder.IdempotencyKey != "" {
		if jsonValue, err := json.Marshal(newOrder); err == nil {
			_ = s.cache.Set(ctx, cacheKey, jsonValue, 30*time.Second)
		}
	}

	slog.InfoContext(ctx, "order created", "order_id", newOrder.ID, "customer_id", newOrder.CustomerID)

	return &orderv1.CreateOrderResponse{Order: mappers.OrderToProto(newOrder)}, nil
}

func (s *orderServer) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, ok := s.orders[req.GetId()]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "order %s not found", req.GetId())
	}

	return &orderv1.GetOrderResponse{Order: mappers.OrderToProto(order)}, nil
}

func (s *orderServer) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, exists := s.orders[req.GetId()]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "order %s not found", req.GetId())
	}

	order.Status = domain.OrderStatus(req.GetStatus().String())

	slog.InfoContext(ctx, "order status updated",
		"order_id", order.ID,
		"new_status", order.Status,
		"request_id", order.RequestID,
	)

	return &orderv1.UpdateOrderStatusResponse{Success: true}, nil
}
