package mtp

import (
	"encoding/json"
	"io"
	"os"

	"github.com/rusq/wipemychat/internal/secure"
)

type credsStorage struct {
	filename   string
	passphrase []byte
}

// creds is the structure of data in the storage.
type creds struct {
	ApiID   secure.Int    `json:"api_id,omitempty"`
	ApiHash secure.String `json:"api_hash,omitempty"`
}

func (cs credsStorage) IsAvailable() bool {
	return cs.filename != "" && len(cs.passphrase) > 0
}

func (cs credsStorage) Save(apiID int, apiHash string) error {
	if err := secure.SetPassphrase(cs.passphrase); err != nil {
		return err
	}
	f, err := os.Create(cs.filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return cs.write(f, apiID, apiHash)
}

func (cs credsStorage) write(f io.Writer, apiID int, apiHash string) error {
	creds := creds{
		ApiID:   secure.Int(apiID),
		ApiHash: secure.String(apiHash),
	}

	enc := json.NewEncoder(f)
	if err := enc.Encode(creds); err != nil {
		return err
	}
	return nil
}

func (cs credsStorage) Load() (int, string, error) {
	if err := secure.SetPassphrase(cs.passphrase); err != nil {
		return 0, "", err
	}
	f, err := os.Open(cs.filename)
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
	return int(creds.ApiID), creds.ApiHash.String(), nil
}
