package api

import (
	"admin-stats/schema"
	"admin-stats/service"
	"net/http"
)

type getGGRQuery struct {
	schema.DateRangeFilter
}

func (a *API) getGrossGamingRevenue(w http.ResponseWriter, r *http.Request) {
	var query getGGRQuery
	if err := a.DecodeQuery(r, &query); err != nil {
		a.respondError(w, err)
		return
	}

	if err := query.DateRangeFilter.Validate(); err != nil {
		a.respondError(w, NewAppError(CodeValidation, err.Error(), http.StatusBadRequest))
		return
	}

	result, err := a.services.Transaction.GetGGR(r.Context(), &service.GetGGRInput{
		From: query.From,
		To:   query.To,
	})
	if err != nil {
		a.respondError(w, err)
		return
	}

	a.respond(w, http.StatusOK, result)
}
