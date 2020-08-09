package provider

import (
	"encoding/json"
	"github.com/voidshard/beancounter/pkg/domain"
)

type token struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}

func ParseTruelayerToken(data []byte) (*domain.Token, error) {
	tkn := &token{}
	err := json.Unmarshal(data, tkn)
	if err != nil {
		return nil, err
	}

	return domain.NewToken(
		tkn.AccessToken,
		tkn.RefreshToken,
		tkn.ExpiresIn,
	), nil
}

type asyncReply struct {
	ResultsURI string `json:"results_uri"`
	Status     string `json:"status"`
	TaskID     string `json:"task_id"`
}

func parseTruelayerAsync(data []byte) (*asyncReply, error) {
	rep := &asyncReply{}
	err := json.Unmarshal(data, rep)
	return rep, err
}

type accountsReply struct {
	Results []tlAccount `json:"results"`
}

type tlAccount struct {
	ID       string     `json:"account_id"`
	Name     string     `json:"display_name"`
	Provider tlProvider `json:"provider"`
}

type tlProvider struct {
	ID   string `json:"provider_id"`
	Name string `json:"display_name"`
}

func parseTruelayerAccounts(data []byte) (*accountsReply, error) {
	// we only parse a small subset of the fields
	rep := &accountsReply{}
	err := json.Unmarshal(data, rep)
	return rep, err
}

type truelayerTransactions struct {
	Results []truelayerTransaction `json:"results"`
}

type truelayerTransaction struct {
	ID             string   `json:"transaction_id"`
	Timestamp      string   `json:"timestamp"`
	Description    string   `json:"description"`
	Amount         float64  `json:"amount"`
	Currency       string   `json:"currency"`
	Type           string   `json:"transaction_type"`
	Category       string   `json:"transaction_category"`
	Classification []string `json:"transaction_classification"`
	Merchant       string   `json:"merchant_name"`
}

func parseTruelayerTransactions(bank, account string, data []byte) ([]*domain.Transaction, error) {
	raw := &truelayerTransactions{}
	err := json.Unmarshal(data, raw)
	if err != nil {
		return nil, err
	}

	txns := []*domain.Transaction{}
	for _, t := range raw.Results {
		txns = append(txns, &domain.Transaction{
			ID:          t.ID,
			Bank:        bank,
			Account:     account,
			Currency:    t.Currency,
			Timestamp:   t.Timestamp,
			Description: t.Description,
			Amount:      t.Amount,
			Type:        t.Type,
			Category:    t.Category,
			Merchant:    t.Merchant,
			Tags:        t.Classification,
		})
	}

	return txns, nil
}
