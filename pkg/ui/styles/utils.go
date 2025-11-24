package styles

import (
	"os"

	"golang.org/x/term"
)

func TerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w < 20 {
		return 80 // fallback
	}
	return w
}
