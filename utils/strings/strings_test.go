package strings

import "testing"

func TestNamedString(t *testing.T) {
	type args struct {
		src string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{
			name: "tt",
			args: args{
				src: "a",
			},
			want: true,
		},
		{
			name: "tt1",
			args: args{
				src: "1",
			},
			want: false,
		},
		{
			name: "tt2",
			args: args{
				src: "Ab",
			},
			want: true,
		},
		{
			name: "tt2",
			args: args{
				src: "_Ab",
			},
			want: false,
		},
		{
			name: "tt2",
			args: args{
				src: "Ab123",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NamedString(tt.args.src); got != tt.want {
				t.Errorf("NamedString() = %v, want %v", got, tt.want)
			}
		})
	}
}
