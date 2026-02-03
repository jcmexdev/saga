package httpx

type CreateOrderRequest struct {
	CustomerID string               `json:"customer_id"`
	Items      []CreateOrderItemDTO `json:"items"`
}

type CreateOrderItemDTO struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type OrderResponse struct {
	ID         string              `json:"id"`
	CustomerID string              `json:"customer_id"`
	Status     string              `json:"status"`
	Total      float64             `json:"total"`
	Reason     string              `json:"reason,omitempty"`
	Items      []OrderItemResponse `json:"items"`
	CreatedAt  string              `json:"created_at"`
	UpdatedAt  string              `json:"updated_at"`
}

type OrderItemResponse struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
