package commands

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"fmt"
	"io"
)

var (
	accent   = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4B5FD")).Bold(true)
	muted    = lipgloss.NewStyle().Foreground(lipgloss.Color("#A5AEC6"))
	mint     = lipgloss.NewStyle().Foreground(lipgloss.Color("#A7F3D0"))
	peach    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FED7AA"))
	rose     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FDA4AF"))
	sky      = lipgloss.NewStyle().Foreground(lipgloss.Color("#BAE6FD"))
	selected = lipgloss.NewStyle().Foreground(lipgloss.Color("#29243B")).Background(lipgloss.Color("#C4B5FD")).Bold(true)
	panel    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#C4B5FD")).Padding(1, 2)
)

func styleInput(input *textinput.Model) {
	styles := input.Styles()
	styles.Focused.Text, styles.Blurred.Text = mint, sky
	styles.Focused.Prompt, styles.Blurred.Prompt = accent, muted
	styles.Focused.Placeholder, styles.Blurred.Placeholder = muted, muted
	input.SetStyles(styles)
}
func success(out io.Writer, message string) { fmt.Fprintln(out, mint.Render("✓ "+safe(message))) }
func storageStatus(out io.Writer, encrypted bool) {
	if encrypted {
		fmt.Fprintln(out, mint.Render("◆ Values encrypted · key names visible · ~/.qv"))
	} else {
		fmt.Fprintln(out, peach.Render("◇ Plain-text storage · not encrypted · ~/.qv"))
	}
}
func PrintError(out io.Writer, err error) {
	fmt.Fprintln(out, rose.Render("✕ ")+rose.Render(safe(err.Error())))
}

func showHelp(out io.Writer) {
	fmt.Fprintln(out, accent.Render("QUICK VAULT"), muted.Render("· Your values, a keystroke away"))
	fmt.Fprintln(out)
	rows := [][2]string{
		{"qv", "Open interactive vault"},
		{"qv set <key> <value>", "Store or replace a value"},
		{"qv get [<key> [--show]]", "Browse keys, or copy a key; --show displays it"},
		{"qv del <key>", "Delete a value"},
		{"qv list", "Select, search, and copy interactively"},
		{"qv init", "First-time setup; optional encryption password"},
		{"qv reset", "Confirm deletion and initialize again"},
		{"qv change-password", "Unlock and re-encrypt with a new password"},
		{"qv help", "Show this help"},
	}
	for _, row := range rows {
		fmt.Fprintln(out, "  "+sky.Width(29).Render(row[0])+muted.Render(row[1]))
	}
	fmt.Fprintln(out, "\n"+peach.Render("Start with qv init. Leave its password empty for plain-text storage."))
	fmt.Fprintln(out, mint.Render("Copied values stay hidden unless you explicitly use --show."))
}
