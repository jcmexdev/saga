package paymentservice

import (
	"context"
	"log"
	"sync"

	paymentv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/payment/v1"
)

type paymentClient struct {
	paymentv1.UnimplementedPaymentServer
	mu       sync.Mutex
	payments map[string]float64
}

func NewClient() *paymentClient {
	return &paymentClient{
		payments: make(map[string]float64),
	}
}
func (s *paymentClient) Charge(ctx context.Context, req *paymentv1.ChargeRequest) (*paymentv1.ChargeResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[Payment] Processing charge for Order: %s, Amount: %.2f", req.OrderId, req.Amount)

	if req.Amount > 500.00 {
		log.Printf("[Payment] Declined: Amount %.2f exceeds limit", req.Amount)
		return &paymentv1.ChargeResponse{
			Success: false, // Esto disparará la compensación en la Saga
		}, nil
	}

	s.payments[req.OrderId] = req.Amount
	log.Printf("[Payment] Charge successful for Order: %s", req.OrderId)

	return &paymentv1.ChargeResponse{
		Success: true,
	}, nil
}

func (s *paymentClient) Refund(ctx context.Context, req *paymentv1.RefundRequest) (*paymentv1.RefundResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	amount, exists := s.payments[req.OrderId]
	if !exists {
		log.Printf("[Payment] Warning: No payment found for order %s to refund", req.OrderId)
		return &paymentv1.RefundResponse{Success: true}, nil
	}

	log.Printf("[Payment] Refunding %.2f for Order: %s", amount, req.OrderId)

	delete(s.payments, req.OrderId)

	return &paymentv1.RefundResponse{
		Success: true,
	}, nil
}
