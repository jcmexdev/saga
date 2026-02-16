package middlewares

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors/constants"
	"google.golang.org/grpc/metadata"
)

// AttachTracingMetadata extracts the request ID (from chi middleware) and
// idempotency key (from the request header) and attaches them to the context
// for propagation to downstream gRPC services.
func AttachTracingMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetReqID(r.Context())
		idempotencyKey := r.Header.Get(constants.HeaderXIdempotencyKey)

		ctx := context.WithValue(r.Context(), constants.ContextKeyRequestID, requestID)
		ctx = context.WithValue(ctx, constants.ContextKeyIdempotencyKey, idempotencyKey)
		ctx = metadata.AppendToOutgoingContext(ctx, constants.HeaderXIdempotencyKey, idempotencyKey)
		ctx = metadata.AppendToOutgoingContext(ctx, constants.HeaderXRequestId, requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
