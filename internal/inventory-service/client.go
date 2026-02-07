package inventoryservice

import (
	"context"
	"log"
	"sync"

	inventoryv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
)

type inventoryClient struct {
	inventoryv1.UnimplementedInventoryServer
	stock        map[string]int32
	mu           sync.Mutex
	reservations map[string][]*inventoryv1.StockItem
}

func NewClient() *inventoryClient {
	return &inventoryClient{
		stock: map[string]int32{
			"prod_1": 15,
			"prod_2": 10,
			"prod_3": 0,
		},
		reservations: make(map[string][]*inventoryv1.StockItem),
	}
}

func (i *inventoryClient) Reserve(ctx context.Context, req *inventoryv1.ReserveRequest) (*inventoryv1.ReserveResponse, error) {
	// 1. Bloqueamos el acceso al mapa para evitar condiciones de carrera (Race Conditions)
	i.mu.Lock()
	defer i.mu.Unlock()

	log.Printf("[Inventory] Processing reservation for Order: %s", req.OrderId)

	// 2. PRIMERA PASADA: Verificar si TODOS los productos tienen stock suficiente
	// Esto es vital: si falta uno solo, no debemos descontar nada todavía (Atomicidad)
	for _, item := range req.Items {
		currentStock, exists := i.stock[item.ProductId]

		if !exists {
			log.Printf("[Inventory] Error: Product %s does not exist", item.ProductId)
			return &inventoryv1.ReserveResponse{Success: false}, nil
		}

		if currentStock < item.Quantity {
			log.Printf("[Inventory] Insufficient stock for %s. Available: %d, Requested: %d",
				item.ProductId, currentStock, item.Quantity)
			return &inventoryv1.ReserveResponse{Success: false}, nil
		}
	}

	// 3. SEGUNDA PASADA: Si llegamos aquí, significa que hay stock de todo.
	// Procedemos a descontar del inventario.
	for _, item := range req.Items {
		i.stock[item.ProductId] -= item.Quantity
		log.Printf("[Inventory] Reserved %d of %s. New stock: %d",
			item.Quantity, item.ProductId, i.stock[item.ProductId])
	}

	// Guardamos qué productos se llevó esta orden
	i.reservations[req.OrderId] = req.Items

	return &inventoryv1.ReserveResponse{Success: true}, nil
}

func (i *inventoryClient) Release(ctx context.Context, req *inventoryv1.ReleaseRequest) (*inventoryv1.ReleaseResponse, error) {
	i.mu.Lock()
	defer i.mu.Unlock() // Garantizamos liberar el mutex al salir del stack

	log.Printf("[Inventory] Compensating (Release) for Order: %s", req.OrderId)

	// 1. Buscamos si tenemos registro de esa orden
	items, exists := i.reservations[req.OrderId]
	if !exists {
		log.Printf("[Inventory] Warning: No reservation found for order %s. Nothing to release.", req.OrderId)
		return &inventoryv1.ReleaseResponse{Success: false}, nil
	}

	// 2. Devolvemos los productos al stock principal
	for _, item := range items {
		i.stock[item.ProductId] += item.Quantity
		log.Printf("[Inventory] Restored %d of %s. New stock: %d",
			item.Quantity, item.ProductId, i.stock[item.ProductId])
	}

	// 3. Limpiamos el mapa de reservas para no gastar memoria
	delete(i.reservations, req.OrderId)

	return &inventoryv1.ReleaseResponse{Success: true}, nil
}
