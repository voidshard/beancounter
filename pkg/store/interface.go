package store

import (
	"github.com/voidshard/beancounter/pkg/domain"
)

type Store interface {
	Write([]*domain.Transaction) error
}
