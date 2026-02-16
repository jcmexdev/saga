package inventoryservice

import (
	"context"
	"log/slog"
	"sync"
	"time"

	inventoryV1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/inventory-service/adapters/grpc/mappers"
	"github.com/jcmexdev/ecommerce-sagas/internal/inventory-service/domain"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/cache"
)

const idempotencyTTL = 60 * time.Second

type inventoryServer struct {
	inventoryV1.UnimplementedInventoryServer
	stock        map[string]int32
	mu           sync.Mutex
	reservations map[string]*domain.Reserve
	cache        cache.Cache
}

var _ inventoryV1.InventoryServer = (*inventoryServer)(nil)

func NewClient(c cache.Cache) *inventoryServer {
	return &inventoryServer{
		stock: map[string]int32{
			"prod_1": 15,
			"prod_2": 10,
			"prod_3": 0,
		},
		reservations: make(map[string]*domain.Reserve),
		cache:        c,
	}
}

func (s *inventoryServer) Reserve(ctx context.Context, req *inventoryV1.ReserveRequest) (*inventoryV1.ReserveResponse, error) {
	newReserve := mappers.StockItemsFromProto(ctx, req)

	if newReserve.IdempotencyKey != "" {
		cacheKey := s.cache.GenerateKey("reserve", newReserve.IdempotencyKey)
		if val, _ := s.cache.Get(ctx, cacheKey); val != "" {
			slog.InfoContext(ctx, "reserve: idempotent response from cache",
				"order_id", req.GetOrderId(),
				"idempotency_key", newReserve.IdempotencyKey,
			)
			return &inventoryV1.ReserveResponse{Success: true}, nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if newReserve.IdempotencyKey != "" {
		for _, existing := range s.reservations {
			if existing.IdempotencyKey == newReserve.IdempotencyKey {
				slog.InfoContext(ctx, "reserve: idempotent response from memory",
					"order_id", req.GetOrderId(),
				)
				return &inventoryV1.ReserveResponse{Success: true}, nil
			}
		}
	}

	slog.InfoContext(ctx, "processing reservation", "order_id", req.GetOrderId())

	for _, item := range newReserve.Items {
		currentStock, exists := s.stock[item.ProductID]
		if !exists {
			slog.WarnContext(ctx, "product not found", "product_id", item.ProductID)
			return &inventoryV1.ReserveResponse{Success: false}, nil
		}
		if currentStock < item.Quantity {
			slog.WarnContext(ctx, "insufficient stock",
				"product_id", item.ProductID,
				"available", currentStock,
				"requested", item.Quantity,
			)
			return &inventoryV1.ReserveResponse{Success: false}, nil
		}
	}

	for _, item := range newReserve.Items {
		s.stock[item.ProductID] -= item.Quantity
		slog.InfoContext(ctx, "stock reserved",
			"product_id", item.ProductID,
			"quantity", item.Quantity,
			"remaining_stock", s.stock[item.ProductID],
		)
	}

	s.reservations[newReserve.OrderID] = newReserve

	if newReserve.IdempotencyKey != "" {
		cacheKey := s.cache.GenerateKey("reserve", newReserve.IdempotencyKey)
		if err := s.cache.Set(ctx, cacheKey, "ok", idempotencyTTL); err != nil {
			// Non-fatal: log and continue. The in-memory map is the fallback.
			slog.WarnContext(ctx, "failed to persist idempotency key to cache",
				"order_id", req.GetOrderId(),
				"error", err,
			)
		}
	}

	return &inventoryV1.ReserveResponse{Success: true}, nil
}

func (s *inventoryServer) Release(ctx context.Context, req *inventoryV1.ReleaseRequest) (*inventoryV1.ReleaseResponse, error) {
	releaseCacheKey := s.cache.GenerateKey("release", req.GetOrderId())
	if val, _ := s.cache.Get(ctx, releaseCacheKey); val != "" {
		slog.InfoContext(ctx, "release: idempotent response from cache", "order_id", req.GetOrderId())
		return &inventoryV1.ReleaseResponse{Success: true}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	slog.InfoContext(ctx, "compensating reservation (release)", "order_id", req.GetOrderId())

	reserve, exists := s.reservations[req.GetOrderId()]
	if !exists {
		slog.WarnContext(ctx, "no reservation found to release", "order_id", req.GetOrderId())
		// Treat as success to keep compensation idempotent.
		return &inventoryV1.ReleaseResponse{Success: true}, nil
	}

	for _, item := range reserve.Items {
		s.stock[item.ProductID] += item.Quantity
		slog.InfoContext(ctx, "stock restored",
			"product_id", item.ProductID,
			"quantity", item.Quantity,
			"new_stock", s.stock[item.ProductID],
		)
	}

	delete(s.reservations, req.GetOrderId())

	// Mark this release as done in Redis to prevent double-release on retries.
	if err := s.cache.Set(ctx, releaseCacheKey, "ok", idempotencyTTL); err != nil {
		slog.WarnContext(ctx, "failed to persist release idempotency key to cache",
			"order_id", req.GetOrderId(),
			"error", err,
		)
	}

	return &inventoryV1.ReleaseResponse{Success: true}, nil
}
