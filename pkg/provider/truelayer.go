package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/voidshard/beancounter/pkg/domain"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// https://docs.truelayer.com/

const (
	retries = 5
)

// check it meets the interface
var _ Provider = &Truelayer{}

func NewTruelayer(clientId, clientSecret string) *Truelayer {
	return &Truelayer{
		clientId:     clientId,
		clientSecret: clientSecret,
	}
}

type Truelayer struct {
	clientId     string
	clientSecret string
}

func (t *Truelayer) OAuthURL(redirect, state string) (string, error) {
	u := &url.URL{Scheme: "https", Host: "auth.truelayer.com"}

	u.Path = "/"

	params := url.Values{}
	params.Add("client_id", t.clientId)
	params.Add("response_type", "code")
	params.Add("redirect_uri", redirect)
	params.Add("state", state)
	params.Add("providers", "uk-oauth-all uk-ob-all")

	// request permission to:
	// get accounts
	// get transactions
	// get balance info for accounts (and with transactions)
	// refresh tokens offline
	params.Add("scope", "balance transactions accounts")

	u.RawQuery = params.Encode() // escape all the things

	return u.String(), nil
}

func (t *Truelayer) Transactions(token *domain.Token, from, to time.Time) ([]*domain.Transaction, error) {
	params := url.Values{}
	params.Add("async", "true")

	u := &url.URL{Scheme: "https", Host: "api.truelayer.com"}
	u.Path = "/data/v1/accounts"
	u.RawQuery = params.Encode()

	result, err := doGet(u.String(), token.Value)
	if err != nil {
		return nil, err
	}

	async, err := parseTruelayerAsync(result)
	if err != nil {
		return nil, err
	}

	return t.pollAccounts(token, from, to, async.ResultsURI)
}

func date(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

func (t *Truelayer) pollAccounts(token *domain.Token, from, to time.Time, poll string) ([]*domain.Transaction, error) {
	sleep(time.Second*120, "giving Truelayer time to fetch accounts")

	result, err := doGet(poll, token.Value)
	if err != nil {
		return nil, err
	}

	accounts, err := parseTruelayerAccounts(result)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("async", "true")
	params.Add("from", date(from))
	params.Add("to", date(to))
	paramQuery := params.Encode() // it's the same for all accounts, no sense redoing this

	wg := &sync.WaitGroup{}

	rChan := make(chan []*domain.Transaction)
	eChan := make(chan error)
	finalChan := make(chan []*domain.Transaction)

	go func() { // error printer
		err := <-eChan
		if err != nil {
			log.Printf("err fetching transactions: %v\n", err)
		}
	}()

	go func() { // fan in
		txns := []*domain.Transaction{}
		for tx := range rChan {
			log.Printf("got %d transactions\n", len(tx))
			txns = append(txns, tx...)
		}
		finalChan <- txns
	}()

	for _, account := range accounts.Results {
		wg.Add(1)

		go func() { // fan out
			defer wg.Done()
			acc := account // closure

			u := &url.URL{Scheme: "https", Host: "api.truelayer.com"}
			u.Path = fmt.Sprintf("/data/v1/accounts/%s/transactions", acc.ID)
			u.RawQuery = paramQuery

			result, err := doGet(u.String(), token.Value)
			if err != nil {
				eChan <- err
				return
			}

			async, err := parseTruelayerAsync(result)
			if err != nil {
				eChan <- err
				return
			}

			tx, err := t.pollTransactions(token, async.ResultsURI, acc.Provider.Name, acc.Name)
			if err != nil {
				eChan <- err
				return
			}

			rChan <- tx
		}()

	}

	wg.Wait()

	close(eChan)
	close(rChan)

	return <-finalChan, nil
}

func (t *Truelayer) pollTransactions(token *domain.Token, poll, bank, acc string) ([]*domain.Transaction, error) {
	sleep(time.Second*120, "giving Truelayer time to fetch transactions")
	result, err := doGet(poll, token.Value)
	if err != nil {
		return nil, err
	}
	return parseTruelayerTransactions(bank, acc, result)
}

func sleep(t time.Duration, msg string) {
	log.Printf("sleeping (%v): %s\n", t, msg)
	time.Sleep(t)
}

func (t *Truelayer) Token(redirect, code string) (*domain.Token, error) {
	// Expect reply like:
	//   {
	//      "access_token": "JWT-ACCESS-TOKEN-HERE",
	//      "expires_in": "JWT-EXPIRY-TIME", // <-- shouldn't this be an int?
	//      "token_type": "Bearer",
	//      "refresh_token": "REFRESH-TOKEN-HERE"
	//   }
	//
	// And no, these are not valid :P
	u := &url.URL{Scheme: "https", Host: "auth.truelayer.com"}
	u.Path = "/connect/token"

	data, err := json.Marshal(map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     t.clientId,
		"client_secret": t.clientSecret,
		"redirect_uri":  redirect,
		"code":          code,
	})
	if err != nil {
		return nil, err
	}

	resp, err := doPost(u.String(), data)
	if err != nil {
		return nil, err
	}

	tok := &token{} // a Truelayer token
	err = json.Unmarshal(resp, tok)
	if err != nil {
		return nil, err
	}

	return domain.NewToken(tok.AccessToken, tok.RefreshToken, tok.ExpiresIn), nil
}

func doGet(uri, token string) ([]byte, error) {
	return doRequest("GET", token, uri, nil)
}

func doPost(uri string, data []byte) ([]byte, error) {
	return doRequest("POST", "", uri, bytes.NewBuffer(data))
}

func doRequest(method, token, uri string, data io.Reader) ([]byte, error) {
	var last error
	client := &http.Client{}

	for i := retries; i > 0; i-- {
		fmt.Println(method, uri)

		req, err := http.NewRequest(method, uri, data)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Content-Type", "application/json")

		if token != "" {
			req.Header.Add("Authorization", fmt.Sprintf("bearer %s", token))
		}

		resp, err := client.Do(req)
		if err != nil {
			last = err
			continue
		}

		body := []byte{}
		if resp.Body != nil {
			defer resp.Body.Close()
			body, err = ioutil.ReadAll(resp.Body)
		}

		status := resp.StatusCode
		if status == http.StatusNoContent && method == "GET" {
			// Truelayer retuns this when we ask about data they're
			// still collecting.
			sleep(time.Second*30, "truelayer returned NoContent, waiting ...")
			return nil, fmt.Errorf("no content returned")
		}
		if status >= 200 && status < 400 {
			// we got ok or a redirect - great!
			return body, nil
		}
		if status >= 500 {
			// they're having trouble, best to retry
			last = fmt.Errorf("got status code: %d (%s)", status, string(body))
			continue
		}

		// ?? probably we screwed up
		return nil, fmt.Errorf("got status code: %d (%s)", status, string(body))
	}

	return nil, last
}
