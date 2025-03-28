package streamserver

import "github.com/gorilla/mux"

func registerHealthCheckRoutes(r *mux.Router) *mux.Router {
	r.HandleFunc("/health", healthCheckRequest)
	return r
}
