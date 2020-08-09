package provider

import (
	"github.com/voidshard/beancounter/pkg/domain"
	"time"
)

type Provider interface {
	Transactions(*domain.Token, time.Time, time.Time) ([]*domain.Transaction, error)
}
