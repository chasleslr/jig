package ui

import (
	"os"

	"github.com/mattn/go-isatty"
)

// IsInteractive returns true if both stdin and stdout are connected to a terminal.
// Use this to gate interactive prompts and maintain backward compatibility with
// piped/scripted usage.
func IsInteractive() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

// IsStdinInteractive returns true if stdin is connected to a terminal.
func IsStdinInteractive() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}

// IsStdoutInteractive returns true if stdout is connected to a terminal.
func IsStdoutInteractive() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}
