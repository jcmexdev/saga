package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/domain/entity"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/ports"
)

type Handler struct {
	orderService ports.OrderService
}

func NewHandler(orderService ports.OrderService) *Handler {
	return &Handler{orderService: orderService}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, ErrorResponse{
		Error:   code,
		Message: msg,
	})
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if req.CustomerID == "" || len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "customer_id and items are required")
		return
	}

	items := make([]entity.CreateOrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		if it.ProductID == "" || it.Quantity <= 0 || it.Price <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_item", "product_id, quantity>0, price>0 are required")
			return
		}
		items = append(items, entity.CreateOrderItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
			Price:     it.Price,
		})
	}

	idempKey := r.Header.Get("X-Idempotency-Key")

	order, err := h.orderService.CreateOrder(r.Context(), req.CustomerID, idempKey, items)
	if err != nil {
		writeError(w, http.StatusBadGateway, "order_service_error", err.Error())
		return
	}

	resp := OrderResponse{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Status:     order.Status,
		Total:      order.Total,
		Reason:     order.Reason,
		Items:      mapItems(order.Items),
		CreatedAt:  order.CreatedAt,
		UpdatedAt:  order.UpdatedAt,
	}

	writeJSON(w, http.StatusCreated, resp)
}

func mapItems(items []entity.CreateOrderItem) []OrderItemResponse {
	out := make([]OrderItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, OrderItemResponse{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
			Price:     it.Price,
		})
	}
	return out
}

func (h *Handler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	orderId := chi.URLParam(r, "id")
	if orderId == "" {
		writeError(w, http.StatusBadRequest, "order_id_required", "")
		return
	}

	order, err := h.orderService.GetOrder(r.Context(), orderId)
	if err != nil {
		writeError(w, http.StatusNotFound, "order_not_found", err.Error())
		return
	}

	resp := OrderResponse{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Status:     order.Status,
		Total:      order.Total,
		Reason:     order.Reason,
		Items:      mapItems(order.Items),
		CreatedAt:  order.CreatedAt,
		UpdatedAt:  order.UpdatedAt,
	}

	writeJSON(w, http.StatusOK, resp)
}
