package api

import (
	"admin-stats/schema"
	"net/http"
)

type recomputeStatsQuery struct {
	schema.DateRangeFilter
}

func (a *API) recomputeStats(w http.ResponseWriter, r *http.Request) {
	var query recomputeStatsQuery
	if err := a.DecodeQuery(r, &query); err != nil {
		a.respondError(w, err)
		return
	}

	if err := query.DateRangeFilter.Validate(); err != nil {
		a.respondError(w, NewAppError(CodeValidation, err.Error(), http.StatusBadRequest))
		return
	}

	if query.From == nil || query.To == nil {
		a.respondError(w, NewAppError(CodeValidation, "from and to are required", http.StatusBadRequest))
		return
	}

	if err := a.services.Stats.Recompute(r.Context(), *query.From, *query.To); err != nil {
		a.respondError(w, err)
		return
	}

	a.respond(w, http.StatusOK, map[string]string{"status": "recomputed"})
}
