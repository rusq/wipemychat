package mtp

import (
	"reflect"
	"testing"
)

func Test_splitBy(t *testing.T) {
	var (
		testInputEven = []int{1, 2, 3, 4, 5}
		testInputOdd  = []int{1, 2, 3, 4, 5, 6}
		testInputSngl = []int{42}
	)
	type args struct {
		n     int
		input []int
		fn    func(i int) int8
	}
	tests := []struct {
		name string
		args args
		want [][]int8
	}{
		{
			"splits as expected (even)",
			args{
				n:     2,
				input: testInputEven,
				fn: func(i int) int8 {
					return int8(testInputEven[i])
				},
			},
			[][]int8{{1, 2}, {3, 4}, {5}},
		},
		{
			"splits as expected (odd)",
			args{
				n:     2,
				input: testInputOdd,
				fn: func(i int) int8 {
					return int8(testInputOdd[i])
				},
			},
			[][]int8{{1, 2}, {3, 4}, {5, 6}},
		},
		{
			"splits as expected (odd)",
			args{
				n:     3,
				input: testInputOdd,
				fn: func(i int) int8 {
					return int8(testInputOdd[i])
				},
			},
			[][]int8{{1, 2, 3}, {4, 5, 6}},
		},
		{
			"splits as expected (empty)",
			args{
				n:     2,
				input: []int{},
				fn: func(i int) int8 {
					return 0
				},
			},
			[][]int8{},
		},
		{
			"splits as expected (one)",
			args{
				n:     2,
				input: testInputSngl,
				fn: func(i int) int8 {
					return int8(testInputSngl[i])
				},
			},
			[][]int8{{42}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitBy(tt.args.n, tt.args.input, tt.args.fn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitBy() = %v, want %v", got, tt.want)
			}
		})
	}
}
