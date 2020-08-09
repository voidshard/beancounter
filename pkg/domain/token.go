package domain

import (
	"time"
)

type Token struct {
	// token value
	Value string `json:"value"`

	// When the token expires in unix time
	Expires int64 `json:"expires"`

	// refresh token, if any
	Refresh string `json:"refresh"`
}

// NewToken creates a new token of the given value with the given Expire time set from a ttl in seconds.
func NewToken(value, refresh string, ttl int) *Token {
	return &Token{
		Value:   value,
		Refresh: refresh,
		Expires: time.Now().UTC().Add(time.Duration(ttl) * time.Second).Unix(),
	}
}

// HasExpired returns if the time now is past Expires
func (t *Token) HasExpired() bool {
	return time.Now().UTC().Unix() >= t.Expires
}
