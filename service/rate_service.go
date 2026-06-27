package service

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
)

type RateService interface {
	ToUSD(ctx context.Context, currency string, amount string) (string, error)
}

type staticRateService struct {
	rates map[string]decimal.Decimal
}

func NewStaticRateService() RateService {
	return &staticRateService{
		rates: map[string]decimal.Decimal{
			"ETH":  decimal.NewFromInt(3000),
			"BTC":  decimal.NewFromInt(65000),
			"USDT": decimal.NewFromInt(1),
		},
	}
}

func (s *staticRateService) ToUSD(_ context.Context, currency string, amount string) (string, error) {
	rate, ok := s.rates[currency]
	if !ok {
		return "", fmt.Errorf("unsupported currency: %s", currency)
	}
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount %q: %w", amount, err)
	}
	return amt.Mul(rate).StringFixed(2), nil
}
