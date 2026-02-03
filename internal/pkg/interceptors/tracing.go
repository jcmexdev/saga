package interceptors

import (
	"context"
	"log"

	"github.com/jcmexdev/ecommerce-sagas/internal/pkg/interceptors/constants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TraceServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, exist := metadata.FromIncomingContext(ctx)
		requestID := ""
		idempotencyID := ""
		if exist {
			if ids := md.Get(constants.HeaderXRequestId); len(ids) > 0 {
				requestID = ids[0]
			}

			if ids := md.Get(constants.HeaderXIdempotencyKey); len(ids) > 0 {
				idempotencyID = ids[0]
			}
		}
		newCtx := context.WithValue(ctx, constants.HeaderXRequestId, requestID)
		newCtx = context.WithValue(newCtx, constants.HeaderXIdempotencyKey, idempotencyID)

		log.Printf("[METHOD][%s][REQUEST_ID][%s][IDEMPOTENCY_KEY][%s]", info.FullMethod, requestID, idempotencyID)

		return handler(newCtx, req)
	}
}
