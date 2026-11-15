package httpx

import (
	"context"

	"github.com/google/uuid"
)

type userCtxKey struct{}

func UserFrom(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userCtxKey{}).(uuid.UUID)
	return id, ok
}

type requestIdCtxKey struct{}

func RequestIdFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIdCtxKey{}).(string)
	return id
}
