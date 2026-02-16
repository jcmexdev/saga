package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/jcmexdev/ecommerce-sagas/internal/api-gateway/infra/httpx/middlewares"
)

// NewRouter builds the HTTP router.
//
// Tracing layer (outermost → innermost):
//
//  1. otelhttp.NewHandler — wraps the entire mux. On every incoming request it:
//     a) Extracts the W3C "traceparent" header (if present, e.g. from a load balancer).
//     b) Creates a root span named after the HTTP route.
//     c) Stores the span in the context so all downstream code can add child spans.
//
//  2. middleware.RequestID — generates a unique request ID (used by chi's Logger).
//
//  3. middlewares.AttachTracingMetadata — copies our custom x-request-id and
//     x-idempotency-key into the context AND into the outgoing gRPC metadata,
//     so they travel alongside the W3C trace headers to every microservice.
func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middlewares.AttachTracingMetadata)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/orders", handler.CreateOrder)
	r.Get("/orders/{id}", handler.GetOrderByID)

	// Wrap the whole mux with otelhttp so every route gets a root span.
	// The span name is set to the matched route pattern (e.g. "POST /orders").
	return otelhttp.NewHandler(r, "api-gateway",
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
	)
}
