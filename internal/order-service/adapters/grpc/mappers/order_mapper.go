package mappers

import (
	"context"
	"time"

	"github.com/google/uuid"

	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/order-service/domain"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors/constants"
)

func OrderFromProto(ctx context.Context, req *orderv1.CreateOrderRequest) *domain.Order {
	if req == nil {
		return nil
	}

	idempKey := interceptors.GetMetadataValue(ctx, constants.HeaderXIdempotencyKey)
	reqID := interceptors.GetMetadataValue(ctx, constants.HeaderXRequestId)

	return &domain.Order{
		ID:             uuid.NewString(),
		CustomerID:     req.GetCustomerId(),
		Items:          mapItemsFromProto(req.GetItems()),
		TotalAmount:    calculateTotal(req.GetItems()),
		Status:         domain.StatusPending,
		IdempotencyKey: idempKey,
		RequestID:      reqID,
		CreatedAt:      time.Now(),
	}
}

func OrderToProto(o *domain.Order) *orderv1.OrderInfo {
	if o == nil {
		return nil
	}

	return &orderv1.OrderInfo{
		Id:          o.ID,
		CustomerId:  o.CustomerID,
		Items:       mapItemsToProto(o.Items),
		TotalAmount: o.TotalAmount,
		Status:      mapStatusToProto(o.Status),
	}
}

func mapItemsFromProto(pbItems []*orderv1.OrderItem) []domain.OrderItem {
	items := make([]domain.OrderItem, len(pbItems))
	for i, item := range pbItems {
		items[i] = domain.OrderItem{
			ProductID: item.GetProductId(),
			Quantity:  int(item.GetQuantity()),
			UnitPrice: item.GetUnitPrice(),
		}
	}
	return items
}

func mapItemsToProto(domainItems []domain.OrderItem) []*orderv1.OrderItem {
	pbItems := make([]*orderv1.OrderItem, len(domainItems))
	for i, item := range domainItems {
		pbItems[i] = &orderv1.OrderItem{
			ProductId: item.ProductID,
			Quantity:  int32(item.Quantity),
			UnitPrice: item.UnitPrice,
		}
	}
	return pbItems
}

func calculateTotal(items []*orderv1.OrderItem) float64 {
	var total float64
	for _, item := range items {
		total += float64(item.GetQuantity()) * item.GetUnitPrice()
	}
	return total
}

func mapStatusToProto(s domain.OrderStatus) orderv1.Status {
	if val, ok := orderv1.Status_value[string(s)]; ok {
		return orderv1.Status(val)
	}
	return orderv1.Status_PENDING
}
