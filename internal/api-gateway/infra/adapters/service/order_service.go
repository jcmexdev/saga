package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/domain/entity"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/ports"
)

type fakeOrderClient struct{}

func NewFakeClient() ports.OrderService {
	return fakeOrderClient{}
}

func (f fakeOrderClient) CreateOrder(ctx context.Context, customerID, idempotencyKey string, items []entity.CreateOrderItem) (*entity.Order, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	total := 0.0
	for _, it := range items {
		total += float64(it.Quantity) * it.Price
	}

	return &entity.Order{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		Status:     "CONFIRMED",
		Total:      total,
		Reason:     "",
		Items:      items,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (f fakeOrderClient) GetOrder(ctx context.Context, id string) (*entity.Order, error) {
	// Para probar GET, si quieres puedes devolver un stub o un error
	panic(" implement me")
	return nil, fmt.Errorf("not implemented in fake client")
}

func (f fakeOrderClient) UpdateStatus(ctx context.Context, id, status string) error {
	panic(" implement me")
}
