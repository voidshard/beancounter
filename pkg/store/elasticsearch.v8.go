package store

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voidshard/beancounter/pkg/domain"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

// from https://github.com/elastic/go-elasticsearch/blob/master/_examples/bulk/indexer.go

const (
	esIndex = "beancounter"
	esFlush = 2048

	envEsAddr = "ELASTICSEARCH_SERVICE_HOST"
	envEsPort = "ELASTICSEARCH_SERVICE_PORT"
)

type ElasticsearchV8 struct {
	addresses []string
}

func NewElasticsearchV8(urls ...string) Store {
	if len(urls) == 0 {
		address := os.Getenv(envEsAddr)
		port := os.Getenv(envEsPort)
		if port == "" {
			port = "9200" // default port
		}
		if address == "" {
			address = "localhost" // default address
		}
		urls = []string{fmt.Sprintf("http://%s:%s", address, port)}
	}

	return &ElasticsearchV8{addresses: urls}
}

func (e *ElasticsearchV8) Write(txns []*domain.Transaction) error {
	retryBackoff := backoff.NewExponentialBackOff()

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: e.addresses,

		// Retry on 429 TooManyRequests statuses
		RetryOnStatus: []int{502, 503, 504, 429},

		// Configure the backoff function
		RetryBackoff: func(i int) time.Duration {
			if i == 1 {
				retryBackoff.Reset()
			}
			return retryBackoff.NextBackOff()
		},

		// Retry up to 5 attempts
		MaxRetries: 5,
	})
	if err != nil {
		return err
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         esIndex,
		FlushBytes:    esFlush,
		Client:        es,
		NumWorkers:    4,
		FlushInterval: 10 * time.Second,
	})
	if err != nil {
		return err
	}

	_, err = es.Indices.Create(esIndex)
	if err != nil {
		log.Println("attempted to make index", esIndex, err)
	}

	for _, t := range txns {
		data, err := t.JSON()
		if err != nil {
			return err
		}

		err = bi.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				// Action field configures the operation to perform (index, create, delete, update)
				Action: "index",

				// DocumentID is the (optional) document ID
				DocumentID: t.ID,

				// Body is an `io.Reader` with the payload
				Body: bytes.NewReader(data),

				// OnSuccess is called for each successful operation
				OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {},

				// OnFailure is called for each failed operation
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						log.Printf("failed to index transactions: %s\n", err)
					} else {
						log.Printf("failed to index transactions %s: %s\n", res.Error.Type, res.Error.Reason)
					}
				},
			},
		)

		if err != nil {
			return err
		}
	}

	err = bi.Close(context.Background())
	if err != nil {
		return nil
	}

	biStats := bi.Stats()
	if biStats.NumFailed > 0 {
		log.Printf("Indexed [%d] documents with [%d] errors\n", int64(biStats.NumFlushed), int64(biStats.NumFailed))
		return fmt.Errorf("failed indexing %d docs", int64(biStats.NumFailed))
	} else {
		log.Printf("Sucessfuly indexed [%d] documents\n", int64(biStats.NumFlushed))
	}

	return nil
}
