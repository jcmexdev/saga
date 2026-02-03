package inventoryservice

import (
	"context"
	"log"
	"sync"

	inventoryV1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
)

type inventoryClient struct {
	inventoryV1.UnimplementedInventoryServer
	stock        map[string]int32
	mu           sync.Mutex
	reservations map[string][]*inventoryV1.StockItem
}

func NewClient() *inventoryClient {
	return &inventoryClient{
		stock: map[string]int32{
			"prod_1": 15,
			"prod_2": 10,
			"prod_3": 0,
		},
		reservations: make(map[string][]*inventoryV1.StockItem),
	}
}

func (i *inventoryClient) Reserve(ctx context.Context, req *inventoryV1.ReserveRequest) (*inventoryV1.ReserveResponse, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	log.Printf("[Inventory] Processing reservation for Order: %s", req.OrderId)

	for _, item := range req.Items {
		currentStock, exists := i.stock[item.ProductId]

		if !exists {
			log.Printf("[Inventory] Error: Product %s does not exist", item.ProductId)
			return &inventoryV1.ReserveResponse{Success: false}, nil
		}

		if currentStock < item.Quantity {
			log.Printf("[Inventory] Insufficient stock for %s. Available: %d, Requested: %d",
				item.ProductId, currentStock, item.Quantity)
			return &inventoryV1.ReserveResponse{Success: false}, nil
		}
	}

	for _, item := range req.Items {
		i.stock[item.ProductId] -= item.Quantity
		log.Printf("[Inventory] Reserved %d of %s. New stock: %d",
			item.Quantity, item.ProductId, i.stock[item.ProductId])
	}

	i.reservations[req.OrderId] = req.Items

	return &inventoryV1.ReserveResponse{Success: true}, nil
}

func (i *inventoryClient) Release(ctx context.Context, req *inventoryV1.ReleaseRequest) (*inventoryV1.ReleaseResponse, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	log.Printf("[Inventory] Compensating (Release) for Order: %s", req.OrderId)

	items, exists := i.reservations[req.OrderId]
	if !exists {
		log.Printf("[Inventory] Warning: No reservation found for order %s. Nothing to release.", req.OrderId)
		return &inventoryV1.ReleaseResponse{Success: false}, nil
	}

	for _, item := range items {
		i.stock[item.ProductId] += item.Quantity
		log.Printf("[Inventory] Restored %d of %s. New stock: %d",
			item.Quantity, item.ProductId, i.stock[item.ProductId])
	}

	delete(i.reservations, req.OrderId)

	return &inventoryV1.ReleaseResponse{Success: true}, nil
}
