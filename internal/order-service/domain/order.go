package domain

import "time"

type Order struct {
	ID             string
	CustomerID     string
	Items          []OrderItem
	TotalAmount    float64
	Status         OrderStatus
	IdempotencyKey string
	RequestID      string
	CreatedAt      time.Time
}

type OrderItem struct {
	ProductID string
	Quantity  int
	UnitPrice float64
}

func (i OrderItem) Subtotal() float64 {
	return float64(i.Quantity) * i.UnitPrice
}

type OrderStatus string

const (
	StatusPending   OrderStatus = "PENDING"
	StatusPaid      OrderStatus = "PAID"
	StatusShipped   OrderStatus = "SHIPPED"
	StatusCancelled OrderStatus = "CANCELLED"
	StatusFailed    OrderStatus = "FAILED"
)
