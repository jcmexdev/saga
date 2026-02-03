package ports

import (
	"context"

	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/domain/entity"
)

type OrderService interface {
	CreateOrder(ctx context.Context, customerID, idempotencyKey string, items []entity.CreateOrderItem) (*entity.Order, error)
	GetOrder(ctx context.Context, id string) (*entity.Order, error)
}
