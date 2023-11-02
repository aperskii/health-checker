package context

import (
	"context"
)

const (
	requestIDKey privateKey = "requestId"
)

type privateKey string

func WithRequestID(ctx context.Context, requestId string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestId)
}

func RequestID(ctx context.Context) string {
	if temp := ctx.Value(requestIDKey); temp != nil {
		if requestID, ok := temp.(string); ok {
			return requestID
		}
	}
	return "undefined"
}
