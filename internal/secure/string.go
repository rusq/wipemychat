package secure

import (
	"bytes"
	"fmt"
)

// String is a type of encrypted string.  Surprise.
type String string

func (es String) String() string {
	return string(es)
}

func (es String) MarshalJSON() ([]byte, error) {
	if len(es) == 0 {
		return []byte(`""`), nil
	}
	data, err := Encrypt(string(es))
	return []byte(`"` + data + `"`), err
}

func (es *String) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, `"`)
	if len(b) == 0 {
		*es = ""
		return nil
	}
	pt, err := Decrypt(string(b))
	if err != nil {
		if err == ErrNotEncrypted {
			*es = String(b)
			return nil
		}
		return fmt.Errorf("%w, while decrypting: %q", err, string(b))
	}
	*es = String(pt)
	return nil
}
