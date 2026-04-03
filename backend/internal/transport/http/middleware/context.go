package middleware

import (
	"context"

	chimw "github.com/go-chi/chi/v5/middleware"
)

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyActor     ctxKey = "actor"
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRequestID).(string)
	if v == "" {
		v = chimw.GetReqID(ctx)
	}
	return v
}

func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, ctxKeyActor, actor)
}

func Actor(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyActor).(string)
	if v == "" {
		return "system"
	}
	return v
}
