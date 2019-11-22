package auth

import (
	"fmt"
	"net/http"
)

type authzMiddlewareMock struct {
	action       string
	resourceType string
}

var _ AuthorizationMiddleware = &authzMiddlewareMock{}

func NewAuthzMiddlewareMock(action, resourceType string) AuthorizationMiddleware {
	return &authzMiddlewareMock{
		action:       action,
		resourceType: resourceType,
	}
}

func (a authzMiddlewareMock) AuthorizeApi(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Mock authz implementation allowing %q on %q for method %q on url %q", a.action, a.resourceType, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}
