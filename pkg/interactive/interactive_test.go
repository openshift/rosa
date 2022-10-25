package interactive

import "testing"

func Test_containsString(t *testing.T) {
	type args struct {
		s     []string
		input string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"contains", args{[]string{"a", "b", "c"}, "a"}, true},
		{"not contains", args{[]string{"a", "b", "c"}, "d"}, false},
		{"empty", args{[]string{}, "d"}, false},
		{"empty", args{[]string{"a", "b", "c"}, ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsString(tt.args.s, tt.args.input); got != tt.want {
				t.Errorf("containsString() %s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
