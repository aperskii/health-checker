package middleware

import (
	"fmt"
	"git.ghpcard.local/csipitca/logcs"
	"github.com/google/uuid"
	"github.com/healthchecker/internal/context"
	"net/http"
)

type RequestID struct {
}

func (mw *RequestID) Set(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		ctx := r.Context()
		ctx = context.WithRequestID(ctx, requestID)
		r = r.WithContext(ctx)
		logcs.Info(fmt.Sprintf("requestID: %s created", requestID))
		next(w, r)
	})
}
