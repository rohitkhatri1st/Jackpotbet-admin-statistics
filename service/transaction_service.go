package service

import (
	"admin-stats/model"
	"admin-stats/repository"
	"admin-stats/server/logger"
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type TransactionService struct {
	repo        repository.TransactionRepository
	log         logger.Logger
	rateService RateService
}

type TransactionServiceOptions struct {
	Repo repository.TransactionRepository
	Log  logger.Logger
}

func NewTransactionService(opts *TransactionServiceOptions) *TransactionService {
	return &TransactionService{
		repo:        opts.Repo,
		log:         opts.Log,
		rateService: NewStaticRateService(),
	}
}

type CreateTransactionInput struct {
	UserID    bson.ObjectID
	RoundID   string
	Type      string
	Currency  string
	Amount    string // pre-validated decimal128 string
	CreatedAt *time.Time
}

type GetTransactionsInput struct {
	From   *time.Time
	To     *time.Time
	Cursor *bson.ObjectID
	Limit  int64
}

type TransactionsResult struct {
	Data   []model.Transaction `json:"data"`
	Cursor *bson.ObjectID      `json:"cursor"`
}

func (s *TransactionService) GetTransactions(ctx context.Context, input *GetTransactionsInput) (*TransactionsResult, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}

	// Fetch one extra to determine if a next page exists.
	transactions, err := s.repo.GetTransactions(ctx, repository.TransactionFilter{
		From:   input.From,
		To:     input.To,
		Cursor: input.Cursor,
		Limit:  input.Limit + 1,
	})
	if err != nil {
		return nil, err
	}

	var nextCursor *bson.ObjectID
	if int64(len(transactions)) > input.Limit {
		transactions = transactions[:input.Limit]
		id := transactions[len(transactions)-1].ID
		nextCursor = &id
	}

	return &TransactionsResult{
		Data:   transactions,
		Cursor: nextCursor,
	}, nil
}

func (s *TransactionService) CreateTransaction(ctx context.Context, input *CreateTransactionInput) (*model.Transaction, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}

	usdAmount, err := s.rateService.ToUSD(ctx, input.Currency, input.Amount)
	if err != nil {
		return nil, err
	}

	// ParseDecimal128 cannot fail — caller must validate the format beforehand.
	amount, _ := bson.ParseDecimal128(input.Amount)
	usd, _ := bson.ParseDecimal128(usdAmount)

	createdAt := time.Now()
	if input.CreatedAt != nil {
		createdAt = *input.CreatedAt
	}

	t := &model.Transaction{
		ID:        bson.NewObjectID(),
		CreatedAt: createdAt,
		UserID:    input.UserID,
		RoundID:   input.RoundID,
		Type:      input.Type,
		Amount:    amount,
		Currency:  input.Currency,
		USDAmount: usd,
	}

	if err := s.repo.CreateTransaction(ctx, *t); err != nil {
		return nil, err
	}
	return t, nil
}

type GetGGRInput struct {
	From *time.Time
	To   *time.Time
}

type GGREntry struct {
	Currency   string `json:"currency"`
	Wagers     string `json:"wagers"`
	Payouts    string `json:"payouts"`
	GGR        string `json:"ggr"`
	WagersUSD  string `json:"wagersUSD"`
	PayoutsUSD string `json:"payoutsUSD"`
	GGRUSD     string `json:"ggrUSD"`
}

type GGRResult struct {
	Data []GGREntry `json:"data"`
}

func (s *TransactionService) GetGGR(ctx context.Context, input *GetGGRInput) (*GGRResult, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}

	totals, err := s.repo.GetGGR(ctx, repository.GGRFilter{
		From: input.From,
		To:   input.To,
	})
	if err != nil {
		return nil, err
	}

	entries := make([]GGREntry, 0, len(totals))
	for _, t := range totals {
		// Convert string amounts to decimal.Decimal for arithmetic, then back to string with fixed precision.
		// Errors are ignored because the repository guarantees valid decimal strings.
		wagers, _ := decimal.NewFromString(t.Wagers)
		payouts, _ := decimal.NewFromString(t.Payouts)
		wagersUSD, _ := decimal.NewFromString(t.WagersUSD)
		payoutsUSD, _ := decimal.NewFromString(t.PayoutsUSD)

		entries = append(entries, GGREntry{
			Currency:   t.Currency,
			Wagers:     wagers.StringFixed(8),
			Payouts:    payouts.StringFixed(8),
			GGR:        wagers.Sub(payouts).StringFixed(8),
			WagersUSD:  wagersUSD.StringFixed(2),
			PayoutsUSD: payoutsUSD.StringFixed(2),
			GGRUSD:     wagersUSD.Sub(payoutsUSD).StringFixed(2),
		})
	}

	return &GGRResult{Data: entries}, nil
}
