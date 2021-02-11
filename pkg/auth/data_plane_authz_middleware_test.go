package auth

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/shared"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDataPlaneAuthzMiddleware_CheckClusterId(t *testing.T) {
	tests := []struct {
		name      string
		token     *jwt.Token
		clusterId string
		want      int
	}{
		{
			name: "should success when clusterId matches",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"kas-fleetshard-operator-cluster-id": "12345",
				},
			},
			clusterId: "12345",
			want:      http.StatusOK,
		},
		{
			name: "should fail when clusterId doesn't match",
			token: &jwt.Token{
				Claims: jwt.MapClaims{
					"kas-fleetshard-operator-cluster-id": "12345",
				},
			},
			clusterId: "invalidid",
			want:      http.StatusNotFound,
		},
		{
			name: "should fail when clusterId claim isn't presented",
			token: &jwt.Token{
				Claims: jwt.MapClaims{},
			},
			clusterId: "12345",
			want:      http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := NewDataPlaneAuthzMiddleware()
			// we need to use mux here to parse the id in the request url
			route := mux.NewRouter().PathPrefix("/agent-cluster/{id}").Subrouter()
			route.HandleFunc("", func(writer http.ResponseWriter, request *http.Request) {
				shared.WriteJSONResponse(writer, http.StatusOK, "")
			}).Methods(http.MethodGet)
			route.Use(func(handler http.Handler) http.Handler {
				return setContextToken(handler, tt.token)
			})
			route.Use(mw.CheckClusterId)
			req := httptest.NewRequest("GET", "http://example.com/agent-cluster/"+tt.clusterId, nil)
			recorder := httptest.NewRecorder()
			route.ServeHTTP(recorder, req)
			status := recorder.Result().StatusCode
			if status != tt.want {
				t.Errorf("expected status code %d but got %d", tt.want, status)
			}
		})
	}

}
