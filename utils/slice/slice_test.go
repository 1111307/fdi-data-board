package slice

import (
	"log"
	"testing"
)

func TestIntersectStringSlice1(t *testing.T) {
	type args struct {
		slicelist [][]int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
		{
			name: "66",
			args: args{
				slicelist: [][]int{
					[]int{1, 2, 3},
					[]int{1},
					[]int{1, 5},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IntersectSliceInt(tt.args.slicelist...)
			log.Println(result)
		})
	}
}

func TestFuzzyContainsString(t *testing.T) {
	type args struct {
		slice  []string
		substr string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
		{
			name: "tt",
			args: args{
				slice:  []string{"hello", "hiworld"},
				substr: "",
			},
			want: nil,
		},
		{
			name: "tt",
			args: args{
				slice:  []string{"hello", "hiworld"},
				substr: "llo",
			},
			want: nil,
		},
		{
			name: "tt",
			args: args{
				slice:  []string{},
				substr: "llo",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Printf("%v", FuzzyContainsString(tt.args.slice, tt.args.substr))
		})
	}
}

func TestCompareSlice(t *testing.T) {
	type args struct {
		s1 []string
		s2 []string
	}
	tests := []struct {
		name  string
		args  args
		want  []string
		want1 []string
	}{
		// TODO: Add test cases.
		{
			name: "tt",
			args: args{
				s1: []string{},
				s2: []string{},
			},
			want:  nil,
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := CompareSlice(tt.args.s1, tt.args.s2)
			log.Println(got)
			log.Println(got1)
		})
	}
}

func TestIntersectSliceString(t *testing.T) {
	type args struct {
		slicelist [][]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				slicelist: [][]string{
					[]string{"hello"},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Println(IntersectSliceString(tt.args.slicelist...))
		})
	}
}

func TestCompareSlice1(t *testing.T) {
	type args struct {
		s1 []string
		s2 []string
	}
	tests := []struct {
		name  string
		args  args
		want  []string
		want1 []string
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				s1: []string{"a"},
				s2: []string{"a", "b"},
			},
			want:  nil,
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := CompareSlice(tt.args.s1, tt.args.s2)
			log.Println(got)
			log.Println(got1)
		})
	}
}
