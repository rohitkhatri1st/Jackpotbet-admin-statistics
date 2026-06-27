package api

import (
	"admin-stats/service"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func (a *API) getTransactions(w http.ResponseWriter, r *http.Request) {
	transactions, err := a.services.Transaction.GetTransactions(r.Context())
	if err != nil {
		a.respondError(w, err)
		return
	}
	a.respond(w, http.StatusOK, transactions)
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

	if err := a.validator.ValidateStruct(req); err != nil {
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
