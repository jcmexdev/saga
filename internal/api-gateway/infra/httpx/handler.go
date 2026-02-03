package httpx

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/domain/entity"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/core/ports"
	"github.com/jcmexdev/ecommerce-sagas/internal/coordinator"
	inventoryv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/inventory/v1"
	orderv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/order/v1"
	paymentv1 "github.com/jcmexdev/ecommerce-sagas/internal/genproto/payment/v1"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors/constants"
)

// Handler handles incoming HTTP requests for the Order domain and coordinates Sagas.
type Handler struct {
	orderService    ports.OrderService  // Local domain service for initial persistence
	orderGrpcClient orderv1.OrderClient // gRPC client to update status via Saga
	paymentClient   paymentv1.PaymentClient
	inventoryClient inventoryv1.InventoryClient
}

// NewHandler initializes the handler with its required domain services and gRPC clients.
func NewHandler(
	os ports.OrderService,
	oc orderv1.OrderClient,
	pc paymentv1.PaymentClient,
	ic inventoryv1.InventoryClient,
) *Handler {
	return &Handler{
		orderService:    os,
		orderGrpcClient: oc,
		paymentClient:   pc,
		inventoryClient: ic,
	}
}

// CreateOrder receives the request, persists a PENDING order, and triggers the Saga.
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
			writeError(w, http.StatusBadRequest, "invalid_item", "product_id, quantity, and price must be valid")
			return
		}
		items = append(items, entity.CreateOrderItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
			Price:     it.Price,
		})
	}

	idempKey := r.Context().Value(constants.HeaderXIdempotencyKey).(string)
	requestId := r.Context().Value(constants.HeaderXRequestId).(string)
	log.Printf("%s", requestId)
	order, err := h.orderService.CreateOrder(r.Context(), req.CustomerID, idempKey, items)
	if err != nil {
		writeError(w, http.StatusBadGateway, "order_service_error", err.Error())
		return
	}
	sagaCtx := context.WithoutCancel(r.Context())
	go h.runOrderSaga(sagaCtx, order)

	writeJSON(w, http.StatusCreated, mapOrderToResponse(order))
}

// GetOrderByID retrieves a single order status by its ID.
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

	writeJSON(w, http.StatusOK, mapOrderToResponse(order))
}

// runOrderSaga manages the distributed transaction across multiple microservices.
func (h *Handler) runOrderSaga(ctx context.Context, order *entity.Order) {
	steps := []coordinator.Step{
		coordinator.NewInventoryStep(h.inventoryClient, order.ID, mapToProtoItems(order.Items)),
		coordinator.NewPaymentStep(h.paymentClient, order.ID, order.Total),
		coordinator.NewConfirmOrderStep(h.orderGrpcClient, order.ID),
	}

	saga := coordinator.NewOrchestrator(steps)

	if err := saga.Start(ctx); err != nil {
		log.Printf("Saga failed for Order %s: %v", order.ID, err)
		_, err = h.orderGrpcClient.UpdateOrderStatus(ctx, &orderv1.UpdateOrderStatusRequest{
			Id:     order.ID,
			Status: orderv1.Status_CANCELLED,
		})
	}
}

// mapToProtoItems converts domain items to Inventory Protobuf items.
func mapToProtoItems(items []entity.CreateOrderItem) []*inventoryv1.StockItem {
	protoItems := make([]*inventoryv1.StockItem, len(items))
	for i, it := range items {
		protoItems[i] = &inventoryv1.StockItem{
			ProductId: it.ProductID,
			Quantity:  int32(it.Quantity),
		}
	}
	return protoItems
}

// mapOrderToResponse converts the internal order entity to the HTTP response format.
func mapOrderToResponse(order *entity.Order) OrderResponse {
	return OrderResponse{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Status:     order.Status,
		Total:      order.Total,
		Reason:     order.Reason,
		Items:      mapItems(order.Items),
		CreatedAt:  order.CreatedAt,
		UpdatedAt:  order.UpdatedAt,
	}
}

func mapItems(items []entity.CreateOrderItem) []OrderItemResponse {
	out := make([]OrderItemResponse, len(items))
	for i, it := range items {
		out[i] = OrderItemResponse{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
			Price:     it.Price,
		}
	}
	return out
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
