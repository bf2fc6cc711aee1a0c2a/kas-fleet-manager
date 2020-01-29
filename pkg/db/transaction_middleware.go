package db

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"

	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/errors"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/logger"
)

// TransactionMiddleware creates a new HTTP middleware that begins a database transaction
// and stores it in the request context.
func TransactionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new Context with the transaction stored in it.
		ctx, err := NewContext(r.Context())
		if err != nil {
			ulog := logger.NewUHCLogger(ctx)
			ulog.Errorf("Could not create transaction: %v", err)
			// use default error to avoid exposing internals to users
			err := errors.GeneralError("")
			operationID := logger.GetOperationID(ctx)
			writeJSONResponse(w, err.HttpCode, err.AsOpenapiError(operationID))
			return
		}

		// Set the value of the request pointer to the value of a new copy of the request with the new context key,vale stored in it
		*r = *r.WithContext(ctx)

		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			hub.ConfigureScope(func(scope *sentry.Scope) {
				txid := ctx.Value("txid").(int64)
				scope.SetTag("db_transaction_id", fmt.Sprintf("%d", txid))
			})
		}

		// Returned from handlers and resolve transactions.
		defer func() { Resolve(r.Context()) }()

		// Continue handling requests.
		next.ServeHTTP(w, r)
	})
}

func writeJSONResponse(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload != nil {
		response, _ := json.Marshal(payload)
		_, _ = w.Write(response)
	}
}
