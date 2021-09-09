package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/generated"
	"github.com/chirino/graphql"
	"github.com/chirino/graphql-4-apis/pkg/apis"
	"github.com/chirino/graphql/graphiql"
	"github.com/chirino/graphql/httpgql"
)

type GraphqlHandler struct {
	Engine   *graphql.Engine
	GraphQL  *httpgql.Handler
	GraphIQL *graphiql.Handler
}

func NewGraphqlHandler(mainHandler http.Handler, graphQLUrl string, apiUrl string) (*GraphqlHandler, error) {

	openapiBytes, err := generated.Asset("connector_mgmt.yaml")
	if err != nil {
		return nil, err
	}
	engine, err := apis.CreateGatewayEngine(apis.Config{
		Openapi: apis.EndpointOptions{
			OpenapiDocument: openapiBytes,
		},
		APIBase: apis.EndpointOptions{
			URL:            apiUrl,
			InsecureClient: true,
		},
		Log: log.New(os.Stderr, "graphql: ", 0),
	})
	if err != nil {
		return nil, err
	}

	return &GraphqlHandler{
		Engine: engine,
		GraphQL: &httpgql.Handler{
			ServeGraphQL:        engine.ServeGraphQL,
			MaxRequestSizeBytes: 1024 * 1024,
		},
		GraphIQL: graphiql.New(graphQLUrl, false),
	}, nil
}

func (h *GraphqlHandler) GetSchema(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(h.Engine.Schema.String()))
}
