package terminalstyle

import (
	"strings"
	"testing"
)

func TestSanitizeOutputPreservesSGRAndRemovesTerminalControls(t *testing.T) {
	input := "\x1b[31mred\x1b[0m\x1b[2J\x1b]0;owned\x07\nnext"
	got := SanitizeOutput(input)
	if got != "\x1b[31mred\x1b[0m\nnext" {
		t.Fatalf("SanitizeOutput() = %q", got)
	}
	if strings.Contains(got, "owned") || strings.Contains(got, "[2J") {
		t.Fatalf("unsafe control content survived: %q", got)
	}
}
