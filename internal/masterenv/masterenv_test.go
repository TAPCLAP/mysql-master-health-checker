package masterenv

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{" True ", true},
		{"0", false},
		{"false", false},
		{"", false},
		{"yes", false},
	}

	for _, tt := range tests {
		if got := Parse(tt.in); got != tt.want {
			t.Errorf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestEnabledUsesEnvironment(t *testing.T) {
	t.Setenv("MYSQL_MASTER", "1")
	if !Enabled() {
		t.Fatal("expected MYSQL_MASTER=1 to be enabled")
	}

	t.Setenv("MYSQL_MASTER", "0")
	if Enabled() {
		t.Fatal("expected MYSQL_MASTER=0 to be disabled")
	}
}
