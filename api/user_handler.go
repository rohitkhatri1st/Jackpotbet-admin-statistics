package api

import (
	"admin-stats/schema"
	"admin-stats/service"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type getWagerPercentileQuery struct {
	schema.DateRangeFilter
}

func (a *API) getWagerPercentile(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(mux.Vars(r)["userId"])
	if err != nil {
		a.respondError(w, NewAppError(CodeValidation, "invalid userId", http.StatusBadRequest))
		return
	}

	var query getWagerPercentileQuery
	if err := a.DecodeQuery(r, &query); err != nil {
		a.respondError(w, err)
		return
	}

	if err := query.DateRangeFilter.Validate(); err != nil {
		a.respondError(w, NewAppError(CodeValidation, err.Error(), http.StatusBadRequest))
		return
	}

	result, err := a.services.Transaction.GetWagerPercentile(r.Context(), &service.GetWagerPercentileInput{
		UserID: userID,
		From:   query.From,
		To:     query.To,
	})
	if err != nil {
		a.respondError(w, err)
		return
	}

	a.respond(w, http.StatusOK, result)
}
