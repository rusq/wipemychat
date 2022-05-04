package secure

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sampleJson = `
{
	"id": 24,
	"username": "hide_ur_pain",
	"secret": "` + encryptedPlainText + `"
}
`

type testType struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Secret   String `json:"secret"`
}

var testStruct = testType{
	ID:       24,
	Username: "hide_ur_pain",
	Secret:   "plain text",
}

func TestString_UnmarshalJSONtoStruct(t *testing.T) {
	z := newTestKeySentinel()
	defer z.Reset()

	var got testType
	err := json.Unmarshal([]byte(sampleJson), &got)
	assert.NoError(t, err, "Unmarshal() unexpected error")
	assert.Equal(t, testStruct, got)
}

func TestString_UnmarshalJSON(t *testing.T) {
	z := newTestKeySentinel()
	defer z.Reset()

	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		args    args
		wantES  String
		wantErr bool
	}{
		{"unencrypted", args{[]byte("unencrypted")}, "unencrypted", false},
		{"unencrypted quoted", args{[]byte("\"unencrypted\"")}, "unencrypted", false},
		{"encrypted", args{[]byte(encryptedPlainText)}, "plain text", false},
		{"encrypted quoted", args{[]byte(`"` + encryptedPlainText + `"`)}, "plain text", false},
		{"invalid", args{[]byte(signature + "i must break you")}, "", true},
		{"empty", args{[]byte{}}, "", false},
		{"empty quoted", args{[]byte(`""`)}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var es String
			if err := es.UnmarshalJSON(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, string(tt.wantES), string(es))
		})
	}
}

func TestString_MarshalJSON(t *testing.T) {
	z := newTestKeySentinel()
	defer z.Reset()

	tests := []struct {
		name    string
		es      String
		want    string
		wantErr bool
	}{
		{"plain text", String("plain text"), "plain text", false},
		{"empty", String(""), ``, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := tt.es.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			encrypted = bytes.Trim(encrypted, `"`)
			if len(encrypted) == 0 && len(tt.want) == 0 {
				return // all good, empty string is an empty string.
			}
			got, err := Decrypt(string(encrypted))
			if err != nil {
				t.Errorf("unexpected decrypt error: %s", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}
