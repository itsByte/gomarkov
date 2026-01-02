package gomarkov

import (
	"reflect"
	"testing"
)

func TestMakePairs(t *testing.T) {
	type args struct {
		tokens []string
		order  int
	}
	tests := []struct {
		name string
		args args
		want []Pair
	}{
		{
			name: "TestMakePairsOne",
			args: args{
				tokens: []string{"a", "b", "c"},
				order:  2,
			},
			want: []Pair{
				{
					CurrentState: []string{"a", "b"},
					NextState:    string("c"),
				},
			},
		},
		{
			name: "TestMakePairsTwo",
			args: args{
				tokens: []string{"a", "b", "c", "d"},
				order:  2,
			},
			want: []Pair{
				{
					CurrentState: []string{"a", "b"},
					NextState:    string("c"),
				},
				{
					CurrentState: []string{"b", "c"},
					NextState:    string("d"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MakePairs(tt.args.tokens, tt.args.order); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakePairs() = %v, want %v", got, tt.want)
			}
		})
	}
}
