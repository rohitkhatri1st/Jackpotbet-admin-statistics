package api

import (
	"admin-stats/server/middleware"
	"net/http"
)

func (a *API) registerRoutes() {
	// May move them to different files if the number of routes grows too large.
	a.openRoutes()
	a.userRoutes()
	a.internalRoutes()
}

// openRoutes registers endpoints that require no authentication.
func (a *API) openRoutes() {
	// add open routes to a.router directly
	r := a.router.PathPrefix("").Subrouter()

	_ = r // remove once routes are added
}

// userRoutes registers endpoints that require a valid user session.
func (a *API) userRoutes() {
	r := a.router.PathPrefix("").Subrouter()
	r.Use(middleware.UserAuth(a.log))

	_ = r // remove once routes are added

}

// internalRoutes registers endpoints restricted to internal/admin callers.
func (a *API) internalRoutes() {
	r := a.router.PathPrefix("/internal").Subrouter()
	r.Use(a.internalAuthMiddleware)

	r.HandleFunc("/transactions", a.getTransactions).Methods(http.MethodGet)
}
