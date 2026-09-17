package terminalstyle

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// SanitizeOutput preserves SGR styling while discarding control sequences that
// could move the cursor or otherwise affect the surrounding terminal UI.
func SanitizeOutput(value string) string {
	var output strings.Builder
	for index := 0; index < len(value); {
		current := value[index]
		if current == 0x1b {
			if index+1 >= len(value) {
				break
			}
			switch value[index+1] {
			case '[':
				end := index + 2
				for end < len(value) && (value[end] < 0x40 || value[end] > 0x7e) {
					end++
				}
				if end < len(value) {
					sequence := value[index : end+1]
					if value[end] == 'm' && validSGR(sequence[2:len(sequence)-1]) {
						output.WriteString(sequence)
					}
					index = end + 1
					continue
				}
				return output.String()
			case ']':
				index += 2
				for index < len(value) {
					if value[index] == 0x07 {
						index++
						break
					}
					if value[index] == 0x1b && index+1 < len(value) && value[index+1] == '\\' {
						index += 2
						break
					}
					index++
				}
				continue
			default:
				index += 2
				continue
			}
		}
		if current < 0x20 && current != '\t' && current != '\n' {
			index++
			continue
		}
		output.WriteByte(current)
		index++
	}
	return output.String()
}

// RenderOutput applies terminal-safety filtering and optionally retains SGR
// styling for an interactive color-capable destination.
func RenderOutput(value string, color bool) string {
	value = SanitizeOutput(value)
	if !color {
		return ansi.Strip(value)
	}
	return value
}

func validSGR(parameters string) bool {
	for _, character := range parameters {
		if (character < '0' || character > '9') && character != ';' && character != ':' {
			return false
		}
	}
	return true
}
