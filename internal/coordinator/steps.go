package coordinator

import (
	"context"
	"fmt"

	inventoryv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	paymentv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/payment/v1"
)

// --- CreateOrderStep ---

type CreateOrderStep struct {
	client  orderv1.OrderClient
	request *orderv1.CreateOrderRequest
	orderID string
}

// NewCreateOrderStep is the constructor for CreateOrderStep
func NewCreateOrderStep(client orderv1.OrderClient, request *orderv1.CreateOrderRequest) *CreateOrderStep {
	return &CreateOrderStep{
		client:  client,
		request: request,
	}
}

func (s *CreateOrderStep) Name() string { return "Create_Order_Step" }

func (s *CreateOrderStep) Execute(ctx context.Context) error {
	res, err := s.client.CreateOrder(ctx, s.request)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}
	s.orderID = res.Order.Id
	return nil
}

func (s *CreateOrderStep) Compensate(ctx context.Context) error {
	_, err := s.client.UpdateOrderStatus(ctx, &orderv1.UpdateOrderStatusRequest{
		Id:     s.orderID,
		Status: orderv1.Status_CANCELLED,
	})
	return err
}

// --- PaymentStep ---

type PaymentStep struct {
	client  paymentv1.PaymentClient
	orderID string
	amount  float64
}

func NewPaymentStep(client paymentv1.PaymentClient, orderID string, amount float64) *PaymentStep {
	return &PaymentStep{
		client:  client,
		orderID: orderID,
		amount:  amount,
	}
}

func (s *PaymentStep) Name() string { return "Payment_Charge_Step" }

func (s *PaymentStep) Execute(ctx context.Context) error {
	res, err := s.client.Charge(ctx, &paymentv1.ChargeRequest{
		OrderId: s.orderID,
		Amount:  s.amount,
	})
	// Check both the gRPC error and the business logic success flag
	if err != nil {
		return fmt.Errorf("payment service error: %w", err)
	}
	if !res.Success {
		return fmt.Errorf("payment declined for order %s", s.orderID)
	}
	return nil
}

func (s *PaymentStep) Compensate(ctx context.Context) error {
	_, err := s.client.Refund(ctx, &paymentv1.RefundRequest{
		OrderId: s.orderID,
	})
	return err
}

// --- InventoryStep ---

type InventoryStep struct {
	client  inventoryv1.InventoryClient
	orderID string
	items   []*inventoryv1.StockItem
}

func NewInventoryStep(client inventoryv1.InventoryClient, orderID string, items []*inventoryv1.StockItem) *InventoryStep {
	return &InventoryStep{
		client:  client,
		orderID: orderID,
		items:   items,
	}
}

func (s *InventoryStep) Name() string { return "Inventory_Reservation_Step" }

func (s *InventoryStep) Execute(ctx context.Context) error {
	res, err := s.client.Reserve(ctx, &inventoryv1.ReserveRequest{
		OrderId: s.orderID,
		Items:   s.items,
	})
	if err != nil {
		return fmt.Errorf("inventory service error: %w", err)
	}
	if !res.Success {
		return fmt.Errorf("inventory insufficient for order %s", s.orderID)
	}
	return nil
}

func (s *InventoryStep) Compensate(ctx context.Context) error {
	_, err := s.client.Release(ctx, &inventoryv1.ReleaseRequest{OrderId: s.orderID})
	return err
}

// --- ConfirmOrderStep ---

type ConfirmOrderStep struct {
	client  orderv1.OrderClient
	orderID string
}

// NewConfirmOrderStep is the constructor for ConfirmOrderStep
func NewConfirmOrderStep(client orderv1.OrderClient, orderID string) *ConfirmOrderStep {
	return &ConfirmOrderStep{
		client:  client,
		orderID: orderID,
	}
}

func (s *ConfirmOrderStep) Name() string { return "Confirm_Order_Step" }

func (s *ConfirmOrderStep) Execute(ctx context.Context) error {
	res, err := s.client.UpdateOrderStatus(ctx, &orderv1.UpdateOrderStatusRequest{
		Id:     s.orderID,
		Status: orderv1.Status_CONFIRMED,
	})
	if err != nil {
		return fmt.Errorf("failed to confirm order gRPC: %w", err)
	}
	if !res.Success {
		return fmt.Errorf("order service refused to confirm order %s", s.orderID)
	}
	return nil
}

func (s *ConfirmOrderStep) Compensate(ctx context.Context) error {
	// Usually empty as it's the last step.
	// In complex systems, you might trigger a 'Return' process here.
	return nil
}
