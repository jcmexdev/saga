package middlewares

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors/constants"
	"google.golang.org/grpc/metadata"
)

func AttachTracingMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId := middleware.GetReqID(r.Context())
		idempotencyKey := r.Header.Get(constants.HeaderXIdempotencyKey)

		ctx := context.WithValue(r.Context(), constants.HeaderXRequestId, requestId)
		ctx = context.WithValue(ctx, constants.HeaderXIdempotencyKey, idempotencyKey)
		ctx = metadata.AppendToOutgoingContext(ctx, constants.HeaderXIdempotencyKey, idempotencyKey)
		ctx = metadata.AppendToOutgoingContext(ctx, constants.HeaderXRequestId, requestId)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
