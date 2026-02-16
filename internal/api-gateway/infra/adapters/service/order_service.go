package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/domain/entity"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/ports"
)

// Ensure fakeOrderService implements the port at compile time.
var _ ports.OrderService = (*fakeOrderService)(nil)

// fakeOrderService is an in-memory implementation of ports.OrderService intended
// for local development and manual testing only. Do NOT use in production.
type fakeOrderService struct{}

// NewFakeOrderService returns an in-memory OrderService for development/testing.
func NewFakeOrderService() ports.OrderService {
	return &fakeOrderService{}
}

func (f *fakeOrderService) CreateOrder(ctx context.Context, customerID, idempotencyKey string, items []entity.CreateOrderItem) (*entity.Order, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	total := 0.0
	for _, it := range items {
		total += float64(it.Quantity) * it.Price
	}

	return &entity.Order{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		Status:     "PENDING",
		Total:      total,
		Items:      items,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (f *fakeOrderService) GetOrder(ctx context.Context, id string) (*entity.Order, error) {
	return nil, fmt.Errorf("GetOrder: not implemented in fake service")
}
