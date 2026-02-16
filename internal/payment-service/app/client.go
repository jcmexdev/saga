package paymentservice

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	paymentv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/payment/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/cache"
)

const idempotencyTTL = 60 * time.Second

type paymentServer struct {
	paymentv1.UnimplementedPaymentServer
	mu       sync.Mutex
	payments map[string]float64
	cache    cache.Cache
}

var _ paymentv1.PaymentServer = (*paymentServer)(nil)

// NewClient creates a new in-memory payment gRPC server backed by a cache for idempotency.
func NewClient(c cache.Cache) *paymentServer {
	return &paymentServer{
		payments: make(map[string]float64),
		cache:    c,
	}
}

func (s *paymentServer) Charge(ctx context.Context, req *paymentv1.ChargeRequest) (*paymentv1.ChargeResponse, error) {
	chargeCacheKey := s.cache.GenerateKey("charge", req.GetOrderId())
	if val, _ := s.cache.Get(ctx, chargeCacheKey); val != "" {
		slog.InfoContext(ctx, "charge: idempotent response from cache", "order_id", req.GetOrderId())
		return &paymentv1.ChargeResponse{Success: true}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, alreadyCharged := s.payments[req.GetOrderId()]; alreadyCharged {
		slog.InfoContext(ctx, "charge: idempotent response from memory", "order_id", req.GetOrderId())
		return &paymentv1.ChargeResponse{Success: true}, nil
	}

	slog.InfoContext(ctx, "processing charge", "order_id", req.GetOrderId(), "amount", req.GetAmount())

	if req.GetAmount() > 500.00 {
		slog.WarnContext(ctx, "charge declined: amount exceeds limit",
			"order_id", req.GetOrderId(),
			"amount", req.GetAmount(),
		)
		return &paymentv1.ChargeResponse{Success: false}, nil
	}

	s.payments[req.GetOrderId()] = req.GetAmount()

	amountStr := fmt.Sprintf("%.2f", req.GetAmount())
	if err := s.cache.Set(ctx, chargeCacheKey, amountStr, idempotencyTTL); err != nil {
		slog.WarnContext(ctx, "failed to persist charge idempotency key to cache",
			"order_id", req.GetOrderId(),
			"error", err,
		)
	}

	slog.InfoContext(ctx, "charge successful", "order_id", req.GetOrderId(), "amount", req.GetAmount())
	return &paymentv1.ChargeResponse{Success: true}, nil
}

func (s *paymentServer) Refund(ctx context.Context, req *paymentv1.RefundRequest) (*paymentv1.RefundResponse, error) {
	refundCacheKey := s.cache.GenerateKey("refund", req.GetOrderId())
	if val, _ := s.cache.Get(ctx, refundCacheKey); val != "" {
		slog.InfoContext(ctx, "refund: idempotent response from cache", "order_id", req.GetOrderId())
		return &paymentv1.RefundResponse{Success: true}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	amount, exists := s.payments[req.GetOrderId()]
	if !exists {
		slog.WarnContext(ctx, "no payment found to refund", "order_id", req.GetOrderId())
		return &paymentv1.RefundResponse{Success: true}, nil
	}

	slog.InfoContext(ctx, "processing refund", "order_id", req.GetOrderId(), "amount", amount)
	delete(s.payments, req.GetOrderId())

	if err := s.cache.Set(ctx, refundCacheKey, "ok", idempotencyTTL); err != nil {
		slog.WarnContext(ctx, "failed to persist refund idempotency key to cache",
			"order_id", req.GetOrderId(),
			"error", err,
		)
	}

	return &paymentv1.RefundResponse{Success: true}, nil
}
