package devops

import "testing"

func TestStripANSI(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no escapes",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "bold reset",
			input: "\x1b[1mhello\x1b[0m",
			want:  "hello",
		},
		{
			name:  "colour codes",
			input: "\x1b[31mred\x1b[39m normal",
			want:  "red normal",
		},
		{
			name:  "256-colour background",
			input: "\x1b[48;5;200mtext\x1b[0m",
			want:  "text",
		},
		{
			name:  "cursor movement",
			input: "\x1b[2A\x1b[4Cupward-left",
			want:  "upward-left",
		},
		{
			name:  "erase line",
			input: "before\x1b[2Kafter",
			want:  "beforeafter",
		},
		{
			name:  "OSC title sequence",
			input: "\x1b]0;window title\x07content",
			want:  "content",
		},
		{
			name:  "two-byte simple sequence",
			input: "\x1bcscreen",
			want:  "screen",
		},
		{
			name:  "mixed output",
			input: "\x1b[32mok\x1b[0m: \x1b[31mfail\x1b[0m",
			want:  "ok: fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StripANSI(tc.input)
			if got != tc.want {
				t.Errorf("StripANSI(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}
