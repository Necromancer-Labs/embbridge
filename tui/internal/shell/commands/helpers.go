package commands

import (
	"fmt"

	"github.com/Necromancer-Labs/embbridge-tui/internal/ui/theme"
)

// PrintError prints a styled error message to stdout.
// Used by commands to display errors in a consistent format.
func PrintError(msg string) {
	fmt.Println(theme.ErrorStyle.Render("Error: " + msg))
}

// PrintSuccess prints a styled success message to stdout.
// Used by commands to confirm successful operations.
func PrintSuccess(msg string) {
	fmt.Println(theme.StatusConnected.Render(msg))
}
