package api

import (
	"admin-stats/schema"
	"admin-stats/service"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	defaultLimit = 20
)

type getTransactionsQuery struct {
	schema.DateRangeFilter
	Cursor string `qs:"cursor"`
	Limit  int64  `qs:"limit" validate:"min=1,max=100"`
}

func (a *API) getTransactions(w http.ResponseWriter, r *http.Request) {
	var query getTransactionsQuery
	if err := a.DecodeQuery(r, &query); err != nil {
		a.respondError(w, err)
		return
	}

	if query.Limit == 0 {
		query.Limit = defaultLimit
	}

	if err := a.validator.Validate(query); err != nil {
		a.respondError(w, NewAppError(CodeValidation, err.Error(), http.StatusBadRequest))
		return
	}

	if err := query.DateRangeFilter.Validate(); err != nil {
		a.respondError(w, NewAppError(CodeValidation, err.Error(), http.StatusBadRequest))
		return
	}

	var cursor *bson.ObjectID
	if query.Cursor != "" {
		id, err := bson.ObjectIDFromHex(query.Cursor)
		if err != nil {
			a.respondError(w, NewAppError(CodeValidation, "invalid cursor", http.StatusBadRequest))
			return
		}
		cursor = &id
	}

	result, err := a.services.Transaction.GetTransactions(r.Context(), &service.GetTransactionsInput{
		From:   query.From,
		To:     query.To,
		Cursor: cursor,
		Limit:  query.Limit,
	})
	if err != nil {
		a.respondError(w, err)
		return
	}
	a.respond(w, http.StatusOK, result)
}

type createTransactionRequest struct {
	// UserID decodes directly to bson.ObjectID — json.Decode fails automatically for invalid hex strings.
	UserID   bson.ObjectID `json:"userId"   validate:"required"`
	RoundID  string        `json:"roundId"  validate:"required"`
	Type     string        `json:"type"     validate:"required,oneof=Wager Payout"`
	Currency string        `json:"currency" validate:"required,oneof=ETH BTC USDT"`
	// Amount stays as string: bson.Decimal128 has no JSON unmarshaler and float loses precision.
	// The decimal128 tag validates the format before parsing.
	Amount string `json:"amount" validate:"required,decimal128"`
	// CreatedAt is optional — defaults to now if omitted. Useful for seeding historical data.
	CreatedAt *time.Time `json:"createdAt"`
}

func (a *API) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req createTransactionRequest
	if err := a.DecodeJSONBody(w, r, &req); err != nil {
		a.respondError(w, err)
		return
	}

	if err := a.validator.Validate(req); err != nil {
		a.respondError(w, NewAppError(CodeValidation, err.Error(), http.StatusBadRequest))
		return
	}

	t, err := a.services.Transaction.CreateTransaction(r.Context(), &service.CreateTransactionInput{
		UserID:    req.UserID,
		RoundID:   req.RoundID,
		Type:      req.Type,
		Currency:  req.Currency,
		Amount:    req.Amount,
		CreatedAt: req.CreatedAt,
	})
	if err != nil {
		a.respondError(w, err, true)
		return
	}

	a.respond(w, http.StatusCreated, t)
}
