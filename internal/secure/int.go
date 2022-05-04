package secure

import (
	"bytes"
	"fmt"
	"strconv"
)

type Int int

func (ei Int) String() string {
	return strconv.Itoa(int(ei))
}

func (ei Int) MarshalJSON() ([]byte, error) {
	data, err := Encrypt(ei.String())
	return []byte(`"` + data + `"`), err
}

func (ei *Int) UnmarshalJSON(b []byte) error {
	b = bytes.Trim(b, `"`)
	if len(b) == 0 {
		*ei = 0
		return nil
	}
	pt, err := Decrypt(string(b))
	if err != nil {
		if err == ErrNotEncrypted {
			val, err := strconv.Atoi(string(b))
			if err != nil {
				return err
			}
			*ei = Int(val)
			return nil
		}
		return fmt.Errorf("%w, while decrypting: %q", err, string(b))
	}
	val, err := strconv.Atoi(pt)
	if err != nil {
		return err
	}
	*ei = Int(val)
	return nil
}
