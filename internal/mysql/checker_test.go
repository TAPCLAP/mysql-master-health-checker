package mysql

import "testing"

func TestParseReadOnly(t *testing.T) {
	tests := []struct {
		in      string
		want    bool
		wantErr bool
	}{
		{"ON", true, false},
		{"off", false, false},
		{"1", true, false},
		{"0", false, false},
		{"true", true, false},
		{"false", false, false},
		{"weird", false, true},
	}

	for _, tt := range tests {
		got, err := ParseReadOnly(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ParseReadOnly(%q) expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseReadOnly(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("ParseReadOnly(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
