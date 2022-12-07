package mtp

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_encryptDecrypt(t *testing.T) {
	var (
		ApiID   = 12345
		ApiHash = "very secure"
	)
	var buf bytes.Buffer
	cs := credsStorage{}
	err := cs.write(&buf, ApiID, ApiHash)
	assert.NoError(t, err)

	gotID, gotHash, gotErr := cs.read(&buf)
	assert.NoError(t, gotErr)
	assert.Equal(t, ApiID, gotID)
	assert.Equal(t, ApiHash, gotHash)

}

func FuzzWriteRead(f *testing.F) {
	type testcase struct {
		id   int
		hash string
	}
	var testcases = []testcase{{12345, "very secure"}, {0, "12345"}, {42, ""}, {-100, "blah"}}
	for _, tc := range testcases {
		f.Add(tc.id, tc.hash)
	}
	cs := credsStorage{}
	f.Fuzz(func(t *testing.T, id int, hash string) {
		var buf bytes.Buffer
		err := cs.write(&buf, id, hash)
		if err != nil {
			return
		}
		gotID, gotHash, gotErr := cs.read(&buf)
		if gotErr != nil {
			return
		}
		assert.Equal(t, id, gotID)
		assert.Equal(t, hash, gotHash)
	})
}
