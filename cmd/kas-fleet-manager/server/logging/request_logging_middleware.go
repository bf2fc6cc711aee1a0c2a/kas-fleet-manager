package logging

import (
	"net/http"
	"time"
)

func RequestLoggingMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		loggingWriter := NewLoggingWriter(writer, request, NewJSONLogFormatter())
		loggingWriter.log(loggingWriter.prepareRequestLog())
		before := time.Now()
		handler.ServeHTTP(loggingWriter, request)
		elapsed := time.Since(before).String()
		loggingWriter.log(loggingWriter.prepareResponseLog(elapsed))
	})
}
