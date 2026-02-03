package interceptors

import (
	"context"
	"log"

	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors/constants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ctxKey string

const RequestIDKey = "x-request-id"
const reqIDKey ctxKey = "request_id"

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		md, ok := metadata.FromIncomingContext(ctx)
		requestID := "unknown"
		idempotencyKey := "unknown"
		if ok {
			log.Printf("VARIABLES %v", md.Get("x-idempotency-key"))
			if ids := md.Get(RequestIDKey); len(ids) > 0 {
				requestID = ids[0]
			}

			if ids := md.Get("x-idempotency-key"); len(ids) > 0 {
				idempotencyKey = ids[0]
			}
		}
		newCtx := context.WithValue(ctx, constants.HeaderXRequestId, requestID)
		newCtx = context.WithValue(newCtx, constants.HeaderXIdempotencyKey, idempotencyKey)

		log.Printf("[%s] Inicio llamada: %s, %s", requestID, idempotencyKey, info.FullMethod)

		return handler(newCtx, req)
	}
}
func GetIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(reqIDKey).(string); ok {
		return id
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ids := md.Get("request_id"); len(ids) > 0 {
			return ids[0]
		}
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		if ids := md.Get("request_id"); len(ids) > 0 {
			return ids[0]
		}
	}

	return "unknown"
}

func ContextWithPropagatedID(ctx context.Context) context.Context {
	id := GetIDFromContext(ctx)
	ctx1 := metadata.AppendToOutgoingContext(ctx, "request_id", id)
	return metadata.AppendToOutgoingContext(ctx1, "x-idempotency-key", GetMetadataValue(ctx, "x-idempotency-key"))
}

func GetMetadataValue(ctx context.Context, key string) string {
	if id, ok := ctx.Value(key).(string); ok {
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
