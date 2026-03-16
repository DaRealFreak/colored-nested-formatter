//go:build !windows

package formatter

import (
	"os"

	"github.com/mattn/go-isatty"
)

// isTerminal checks if we are currently in a terminal
func (h *Handler) isTerminal() bool {
	switch out := h.writer.(type) {
	case *os.File:
		return isatty.IsTerminal(out.Fd()) || isatty.IsCygwinTerminal(out.Fd())
	default:
		return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	}
}
