package server

import (
	environments2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

type Server interface {
	Start()
	Stop() error
	Listen() (net.Listener, error)
	Serve(net.Listener)
}

func removeTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		next.ServeHTTP(w, r)
	})
}

// Exit on error
func check(err error, msg string) {
	if err != nil && err != http.ErrServerClosed {
		glog.Errorf("%s: %s", msg, err)
		sentry.CaptureException(err)
		sentry.Flush(environments2.Environment().Config.Sentry.Timeout)
		os.Exit(1)
	}
}
