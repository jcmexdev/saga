package interceptors

import (
	"context"
	"log/slog"

	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors/constants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TraceServerInterceptor is a gRPC unary server interceptor that extracts
// tracing metadata (request ID and idempotency key) from incoming gRPC metadata
// and propagates them into the request context.
func TraceServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		requestID := ""
		idempotencyKey := ""
		if ok {
			if ids := md.Get(constants.HeaderXRequestId); len(ids) > 0 {
				requestID = ids[0]
			}
			if ids := md.Get(constants.HeaderXIdempotencyKey); len(ids) > 0 {
				idempotencyKey = ids[0]
			}
		}

		newCtx := context.WithValue(ctx, constants.ContextKeyRequestID, requestID)
		newCtx = context.WithValue(newCtx, constants.ContextKeyIdempotencyKey, idempotencyKey)

		slog.InfoContext(newCtx, "gRPC request",
			"method", info.FullMethod,
			"request_id", requestID,
			"idempotency_key", idempotencyKey,
		)

		return handler(newCtx, req)
	}
}

// GetMetadataValue retrieves a metadata value from the context by key.
// It checks the context value first, then falls back to gRPC incoming/outgoing metadata.
func GetMetadataValue(ctx context.Context, key string) string {
	if id, ok := ctx.Value(constants.ContextKeyRequestID).(string); ok && key == constants.HeaderXRequestId {
		return id
	}
	if id, ok := ctx.Value(constants.ContextKeyIdempotencyKey).(string); ok && key == constants.HeaderXIdempotencyKey {
		return id
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ids := md.Get(key); len(ids) > 0 {
			return ids[0]
		}
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if ids := md.Get(key); len(ids) > 0 {
			return ids[0]
		}
	}

	return ""
}
