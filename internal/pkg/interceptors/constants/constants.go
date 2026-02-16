package constants

// contextKey is an unexported type for context keys in this package.
// Using a custom type prevents collisions with keys from other packages
// that might use the same underlying string value.
type contextKey string

const (
	HeaderXRequestId      = "x-request-id"
	HeaderXIdempotencyKey = "x-idempotency-key"

	// ContextKeyRequestID is the context key for the request ID.
	ContextKeyRequestID contextKey = HeaderXRequestId
	// ContextKeyIdempotencyKey is the context key for the idempotency key.
	ContextKeyIdempotencyKey contextKey = HeaderXIdempotencyKey
)
