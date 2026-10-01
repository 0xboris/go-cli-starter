package iostreams

import "testing"

func TestCanPrompt(t *testing.T) {
	tests := []struct {
		name                  string
		stdin, stdout, stderr bool
		never                 bool
		want                  bool
	}{
		{"all terminals", true, true, true, false, true},
		{"stdout redirected", true, false, true, false, true},
		{"stderr redirected", true, true, false, false, false},
		{"stdin piped", false, true, true, false, false},
		{"--no-input", true, true, true, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := Test()
			ios.SetStdinTTY(tt.stdin)
			ios.SetStdoutTTY(tt.stdout)
			ios.SetStderrTTY(tt.stderr)
			ios.SetNeverPrompt(tt.never)
			if got := ios.CanPrompt(); got != tt.want {
				t.Errorf("CanPrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}
