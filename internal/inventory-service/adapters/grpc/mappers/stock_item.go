package mappers

import (
	"context"

	inventoryv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/inventory-service/domain"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors/constants"
)

func StockItemsFromProto(ctx context.Context, req *inventoryv1.ReserveRequest) *domain.Reserve {
	return &domain.Reserve{
		OrderID:        req.GetOrderId(),
		Items:          mapItemsFromProto(req.GetItems()),
		IdempotencyKey: interceptors.GetMetadataValue(ctx, constants.HeaderXIdempotencyKey),
		RequestID:      interceptors.GetMetadataValue(ctx, constants.HeaderXRequestId),
	}
}

func mapItemsFromProto(protoItems []*inventoryv1.StockItem) []*domain.StockItem {
	items := make([]*domain.StockItem, len(protoItems))
	for i, protoItem := range protoItems {
		items[i] = &domain.StockItem{
			ProductID: protoItem.GetProductId(),
			Quantity:  protoItem.GetQuantity(),
		}
	}
	return items
}
