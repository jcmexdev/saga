package service

import (
	"context"
	"fmt"

	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"

	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/domain/entity"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/ports"
)

// GRPCOrderService es el adapter que habla con el OrderService gRPC
type GRPCOrderService struct {
	client orderv1.OrderClient
}

// NewGRPCOrderClient devuelve AL PORT, como tú querías.
func NewGRPCOrderClient(client orderv1.OrderClient) ports.OrderService {
	return &GRPCOrderService{client: client}
}

// Aseguramos en compile-time que implementa la interfaz
var _ ports.OrderService = (*GRPCOrderService)(nil)

// CreateOrder implementa el puerto usando gRPC por debajo.
func (s *GRPCOrderService) CreateOrder(
	ctx context.Context,
	customerID, idempotencyKey string,
	items []entity.CreateOrderItem,
) (*entity.Order, error) {
	protoItems := make([]*orderv1.OrderItem, 0, len(items))

	for _, it := range items {
		protoItems = append(protoItems, &orderv1.OrderItem{
			ProductId: it.ProductID,
			Quantity:  int32(it.Quantity),
			UnitPrice: it.Price,
		})
	}

	req := &orderv1.CreateOrderRequest{
		CustomerId: customerID,
		Items:      protoItems,
	}

	res, err := s.client.CreateOrder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc CreateOrder: %w", err)
	}

	po := res.GetOrder()
	if po == nil {
		return nil, fmt.Errorf("grpc CreateOrder: empty order in response")
	}

	return mapProtoOrderToEntity(po), nil
}

// GetOrder implementa el puerto usando gRPC
func (s *GRPCOrderService) GetOrder(ctx context.Context, id string) (*entity.Order, error) {
	req := &orderv1.GetOrderRequest{Id: id}

	res, err := s.client.GetOrder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("grpc GetOrder: %w", err)
	}

	po := res.GetOrder()
	if po == nil {
		return nil, fmt.Errorf("grpc GetOrder: empty order in response")
	}

	return mapProtoOrderToEntity(po), nil
}

func mapProtoOrderToEntity(po *orderv1.OrderInfo) *entity.Order {
	return &entity.Order{
		ID:         po.GetId(),
		CustomerID: po.GetCustomerId(),
		Status:     po.GetStatus().String(),
		Total:      po.GetTotalAmount(),
		Reason:     "",
		Items:      mapProtoItemsToEntity(po.GetItems()),
		CreatedAt:  "",
		UpdatedAt:  "",
	}
}

func mapProtoItemsToEntity(items []*orderv1.OrderItem) []entity.CreateOrderItem {
	out := make([]entity.CreateOrderItem, 0, len(items))
	for _, it := range items {
		out = append(out, entity.CreateOrderItem{
			ProductID: it.GetProductId(),
			Quantity:  int(it.GetQuantity()),
			Price:     it.GetUnitPrice(),
		})
	}
	return out
}
