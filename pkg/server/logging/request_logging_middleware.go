package logging

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/logger"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

func RequestLoggingMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		logEvent := logger.NewLogEventFromString(mux.CurrentRoute(request).GetName())
		action := logEvent.Type
		ctx := request.Context()
		if action != "" {
			ctx = context.WithValue(request.Context(), logger.ActionKey, action)
		}
		ctx = context.WithValue(ctx, logger.RemoteAddrKey, request.RemoteAddr)
		request = request.WithContext(ctx)

		loggingWriter := NewLoggingWriter(writer, request, NewJSONLogFormatter())
		loggingWriter.Log(loggingWriter.prepareRequestLog())
		before := time.Now()
		handler.ServeHTTP(loggingWriter, request)
		elapsed := time.Since(before).String()
		loggingWriter.Log(loggingWriter.prepareResponseLog(elapsed))
	})
}
