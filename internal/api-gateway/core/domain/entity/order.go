package entity

type CreateOrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

type Order struct {
	ID         string
	CustomerID string
	Status     string
	Total      float64
	Reason     string
	Items      []CreateOrderItem
	CreatedAt  string
	UpdatedAt  string
}
