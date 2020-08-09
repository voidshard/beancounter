package store

import (
	"encoding/json"
	"github.com/voidshard/beancounter/pkg/domain"
	"io/ioutil"
)

type JSONFile struct {
	filename string
}

func NewJSONFile(filename string) Store {
	return &JSONFile{filename: filename}
}

func (f *JSONFile) Write(txns []*domain.Transaction) error {
	data, err := json.Marshal(txns)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(f.filename, data, 0644)
}
