package cmdutil

import "testing"

func TestMergePath(t *testing.T) {
	tests := []struct {
		name    string
		current string
		user    string
		machine string
		sep     string
		want    string
	}{
		{name: "dedup and order", current: "a;b", user: "b;c", machine: "c;d", sep: ";", want: "a;b;c;d"},
		{name: "empty scopes", current: "a;b", user: "", machine: "", sep: ";", want: "a;b"},
		{name: "drops empty entries", current: "a;;", user: "", machine: ";d", sep: ";", want: "a;d"},
		{name: "all scopes merge", current: "", user: "usr", machine: "sys", sep: ";", want: "usr;sys"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergePath(tt.current, tt.user, tt.machine, tt.sep); got != tt.want {
				t.Errorf("MergePath() = %q, want %q", got, tt.want)
			}
		})
	}
}