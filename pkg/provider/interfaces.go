package provider

import (
	"github.com/gorilla/mux"
)

type RouteLoader interface {
	AddRoutes(mainRouter *mux.Router) error
}
