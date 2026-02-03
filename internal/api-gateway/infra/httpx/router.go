package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/infra/httpx/middlewares"
)

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middlewares.AttachTracingMetadata)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/orders", handler.CreateOrder)
	r.Get("/orders/{id}", handler.GetOrderByID)
	return r
}
