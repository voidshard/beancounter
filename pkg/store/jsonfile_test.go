package store

import (
	"github.com/stretchr/testify/assert"
	"github.com/voidshard/beancounter/pkg/domain"
	"testing"
)

func TestWrite(t *testing.T) {
	jf := NewJSONFile("/tmp/test.json")

	err := jf.Write([]*domain.Transaction{
		&domain.Transaction{ID: "1"},
		&domain.Transaction{ID: "2"},
	})

	assert.Nil(t, err)
}
