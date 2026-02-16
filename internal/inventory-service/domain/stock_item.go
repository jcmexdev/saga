package domain

type Reserve struct {
	OrderID        string
	Items          []*StockItem
	IdempotencyKey string
	RequestID      string
}

type StockItem struct {
	ProductID string
	Quantity  int32
}
