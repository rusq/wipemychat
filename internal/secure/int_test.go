package secure

import (
	"math"
	"testing"
)

func TestInt_MarshalUnmarshalJSON(t *testing.T) {
	s := newTestKeySentinel()
	defer s.Reset()

	testcases := []int{123, -1, math.MaxInt}
	for _, tc := range testcases {
		val := Int(tc)
		data, err := val.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		var got Int
		if err := got.UnmarshalJSON(data); err != nil {
			t.Fatal(err)
		}
	}
}

func FuzzMarshalUnmarshalJSON(f *testing.F) {
	s := newTestKeySentinel()
	defer s.Reset()

	testcases := []int{123, -1, math.MaxInt}
	for _, tc := range testcases {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, input int) {
		val := Int(input)
		data, err := val.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		var got Int
		if err := got.UnmarshalJSON(data); err != nil {
			t.Fatal(err)
		}
	})
}
