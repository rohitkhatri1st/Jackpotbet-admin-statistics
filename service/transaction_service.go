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

// ---------- Wager Percentile ----------------------------------------------------

type GetWagerPercentileInput struct {
	UserID bson.ObjectID
	From   *time.Time
	To     *time.Time
}

type WagerPercentileResult struct {
	UserID     string `json:"userId"`
	TotalUSD   string `json:"totalUSD"`
	Rank       int    `json:"rank"`
	TotalUsers int    `json:"totalUsers"`
	Percentile string `json:"percentile"` // top N%, e.g. "2.00"
}

func (s *TransactionService) GetWagerPercentile(ctx context.Context, input *GetWagerPercentileInput) (*WagerPercentileResult, error) {
	if input == nil {
		return nil, errors.New("input must not be nil")
	}

	allTotals, err := s.repo.GetAllUserWagerTotals(ctx, repository.WagerRankFilter{
		From: input.From,
		To:   input.To,
	})
	if err != nil {
		return nil, err
	}

	totalUsers := len(allTotals)
	userRank := 0
	userTotalUSD := decimal.Zero

	for i, t := range allTotals {
		if t.UserID == input.UserID {
			userRank = i + 1
			userTotalUSD, _ = decimal.NewFromString(t.TotalUSD)
			break
		}
	}

	// User had no wagers in this period — place them last.
	if userRank == 0 {
		totalUsers++
		userRank = totalUsers
	}

	percentile := decimal.NewFromInt(int64(userRank)).
		Div(decimal.NewFromInt(int64(totalUsers))).
		Mul(decimal.NewFromInt(100))

	return &WagerPercentileResult{
		UserID:     input.UserID.Hex(),
		TotalUSD:   userTotalUSD.StringFixed(2),
		Rank:       userRank,
		TotalUsers: totalUsers,
		Percentile: percentile.StringFixed(2),
	}, nil
}
