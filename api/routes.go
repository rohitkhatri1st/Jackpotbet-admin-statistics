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
// currently this uses a static token as per assignment's requirement, but could be swapped for a more robust auth system in the future.
func (a *API) internalRoutes() {
	r := a.router.PathPrefix("/internal").Subrouter()
	r.Use(a.internalAuthMiddleware)

	r.HandleFunc("/transactions", a.getTransactions).Methods(http.MethodGet)
	r.HandleFunc("/transactions", a.createTransaction).Methods(http.MethodPost)
	r.HandleFunc("/gross_gaming_rev", a.getGrossGamingRevenue).Methods(http.MethodGet)
	r.HandleFunc("/daily_wager_volume", a.getDailyWagerVolume).Methods(http.MethodGet)
	r.HandleFunc("/user/{userId}/wager_percentile", a.getWagerPercentile).Methods(http.MethodGet)
}
