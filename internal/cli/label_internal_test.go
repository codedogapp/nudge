package cli

import "testing"

func TestTheMachineLabelsTheProject(t *testing.T) {
	tests := []struct{ project, host, want string }{
		{"atlas", "work-laptop", "atlas (work-laptop)"},
		{"atlas", "bakh.local", "atlas (bakh)"},
		{"atlas", "", "atlas"},
		{"atlas", ".local", "atlas"},
	}

	for _, test := range tests {
		if got := labelled(test.project, test.host); got != test.want {
			t.Errorf("labelled(%q, %q) = %q, want %q", test.project, test.host, got, test.want)
		}
	}
}
