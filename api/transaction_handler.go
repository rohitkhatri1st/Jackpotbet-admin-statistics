package api

import (
	"encoding/json"
	"net/http"
)

func (a *API) getTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := a.services.Transaction.GetTransactions(r.Context())
	if err != nil {
		a.log.Error(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}
