package ctxutil

import "context"

type key[T any] struct{}

func WithValue[T any](ctx context.Context, t T) context.Context {
	return context.WithValue(ctx, key[T]{}, t)
}

func Value[T any](ctx context.Context) (T, bool) {
	t, ok := ctx.Value(key[T]{}).(T)
	return t, ok
}
