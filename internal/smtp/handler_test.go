package smtp

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name, in string
		max      int
		want     string
	}{
		{"short", "hello", 10, "hello"},
		{"cuts by characters", "héllo wörld", 5, "héllo"},
		{"drops NUL", "a\x00b", 10, "ab"},
		{"replaces invalid UTF-8", "a\xffb", 10, "a�b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncate(tt.in, tt.max); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}

func TestDecodeHeader(t *testing.T) {
	if got := decodeHeader("=?UTF-8?B?Q29uZmlybWF0aW9u?="); got != "Confirmation" {
		t.Errorf("decodeHeader = %q", got)
	}
	if got := decodeHeader("plain subject"); got != "plain subject" {
		t.Errorf("decodeHeader = %q", got)
	}
}
