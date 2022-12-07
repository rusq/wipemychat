package mtp

import (
	"encoding/json"
	"io"

	"github.com/rusq/encio"
)

type credsStorage struct {
	filename string
}

// creds is the structure of data in the storage.
type creds struct {
	ApiID   int    `json:"api_id,omitempty"`
	ApiHash string `json:"api_hash,omitempty"`
}

func (cs credsStorage) IsAvailable() bool {
	return cs.filename != ""
}

func (cs credsStorage) Save(apiID int, apiHash string) error {
	f, err := encio.Create(cs.filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return cs.write(f, apiID, apiHash)
}

func (cs credsStorage) write(f io.Writer, apiID int, apiHash string) error {
	creds := creds{
		ApiID:   apiID,
		ApiHash: apiHash,
	}

	enc := json.NewEncoder(f)
	if err := enc.Encode(creds); err != nil {
		return err
	}
	return nil
}

func (cs credsStorage) Load() (int, string, error) {
	f, err := encio.Open(cs.filename)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	return cs.read(f)
}

func (cs credsStorage) read(r io.Reader) (int, string, error) {
	var creds creds
	dec := json.NewDecoder(r)
	if err := dec.Decode(&creds); err != nil {
		return 0, "", err
	}
	return creds.ApiID, creds.ApiHash, nil
}
