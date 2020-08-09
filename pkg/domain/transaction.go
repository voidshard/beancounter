package domain

import (
	"encoding/json"
)

type Transaction struct {
	ID string `json:"id"`

	Bank    string `json:"bank"`
	Account string `json:"account"`

	Currency    string  `json:"currency"`
	Timestamp   string  `json:"timestamp"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Category    string  `json:"category"`
	Merchant    string  `json:"merchant"`

	Tags []string `json:"tags"`
}

func (t *Transaction) JSON() ([]byte, error) {
	return json.Marshal(t)
}
