/*OAuth flow support*/
package main

import (
	"encoding/base64"
	"fmt"
	"github.com/voidshard/beancounter/pkg/crypto"
	"github.com/voidshard/beancounter/pkg/domain"
	"github.com/voidshard/beancounter/pkg/provider"
	"github.com/voidshard/beancounter/pkg/store"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type oauthState struct {
	Nonce         string `json:"nonce"`
	keyEncryption string `json:"-"`
	keySignature  string `json:"-"`
}

func (s *oauthState) Verify(blob string) bool {
	// we actually don't care about the value (we know it) only that
	// the message is actually from Truelayer.
	_, err := crypto.Decrypt(blob, s.keyEncryption, s.keySignature)
	return err == nil
}

func (s *oauthState) Encrypt() (string, error) {
	safenonce := base64.StdEncoding.EncodeToString([]byte(s.Nonce))
	return crypto.Encrypt([]byte(safenonce), s.keyEncryption, s.keySignature)
}

func NewState() (*oauthState, error) {
	enckey, err := crypto.NewRandomKey()
	if err != nil {
		return nil, err
	}

	signkey, err := crypto.NewRandomKey()

	return &oauthState{
		Nonce:         fmt.Sprintf("%d", time.Now().UnixNano()),
		keyEncryption: enckey,
		keySignature:  signkey,
	}, err
}

func getStore(out string) (store.Store, error) {
	bits := strings.SplitN(out, ":", 2)
	if len(bits) != 2 {
		return nil, fmt.Errorf("invalid out path, expected [jsonfile:/path/to/file.json] or [es8:http://elasticsearch:9200]")
	}

	if bits[0] == "es8" {
		return store.NewElasticsearchV8(bits[1]), nil
	}

	return store.NewJSONFile(bits[1]), nil
}

func (l *truelayerCmd) Run(ctx *context) error {
	u, err := url.Parse(l.Redirect)
	if err != nil {
		return err
	}
	u.Path = ""

	storage, err := getStore(l.Out)
	if err != nil {
		return err
	}

	state, err := NewState()
	if err != nil {
		return err
	}

	// make oauth url
	tl := provider.NewTruelayer(l.TruelayerClientId, l.TruelayerSecret)
	cypher, err := state.Encrypt()
	if err != nil {
		return err
	}
	oauth, err := tl.OAuthURL(u.String(), cypher)
	if err != nil {
		return err
	}

	// set up a listener
	incoming := make(chan *domain.Token)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			return // *sigh*
		}

		fmt.Println("recieved message at:", r.URL.String())
		tkn, err := processCodeRequest(tl, l.Redirect, state, r)
		if err != nil {
			panic(fmt.Sprintf("failed to get token: %v", err))
		}
		incoming <- tkn
		w.WriteHeader(200)
	})
	go http.ListenAndServe(fmt.Sprintf(":%d", l.Port), nil)

	// prompt user, block and wait for reply from truelayer
	fmt.Println("Go to:", oauth)
	tkn := <-incoming
	if tkn == nil {
		return nil
	}

	fmt.Println("Fetching transactions")
	txns, err := tl.Transactions(tkn, time.Now().AddDate(0, 0, -1*l.Days), time.Now())
	if err != nil {
		return err
	}

	fmt.Println("Writing to", l.Out)
	return storage.Write(txns)
}

func processCodeRequest(tl *provider.Truelayer, redirect string, state *oauthState, r *http.Request) (*domain.Token, error) {
	// read out our code
	qmap := r.URL.Query()

	blob, ok := qmap["state"]
	if !ok || len(blob) == 0 {
		return nil, fmt.Errorf("state not returned")
	}
	if !state.Verify(blob[0]) {
		return nil, fmt.Errorf("failed to decrypt state & assert signature")
	}

	// finally, we can get our code
	code, ok := r.URL.Query()["code"]
	if !ok || len(code) == 0 {
		return nil, fmt.Errorf("code not returned")
	}
	fmt.Println("message verified, exchanging code for token with Truelayer")

	// which we use to get a token ..
	return tl.Token(redirect, code[0])
}
