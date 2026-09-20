package commands

import (
	"errors"
	"io"
	"os"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
)

var errCancelled = errors.New("cancelled; vault has not been changed")

type promptModel struct {
	title, hint     string
	input           textinput.Model
	done, cancelled bool
	width           int
}

func (m promptModel) Init() tea.Cmd { return m.input.Focus() }
func (m promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(12, min(68, msg.Width-8))
		m.input.SetWidth(max(4, m.width-6))
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
func (m promptModel) View() tea.View {
	if m.done || m.cancelled {
		return tea.NewView("")
	}
	return tea.NewView(panel.Width(m.width).Render(accent.Render(m.title) + "\n\n" + peach.Render(m.hint) + "\n\n" + m.input.View() + "\n\n" + muted.Render("enter continue · esc cancel")))
}

func ask(out io.Writer, title, hint string, secret bool) (string, error) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		return "", errors.New("this command requires an interactive terminal")
	}
	input := textinput.New()
	input.SetVirtualCursor(true)
	input.SetWidth(48)
	styleInput(&input)
	input.Focus()
	if secret {
		input.EchoMode = textinput.EchoPassword
		input.EchoCharacter = '•'
	}
	result, err := tea.NewProgram(promptModel{title: title, hint: hint, input: input, width: 60}, tea.WithInput(os.Stdin), tea.WithOutput(out)).Run()
	if err != nil {
		return "", err
	}
	m := result.(promptModel)
	if !m.done || m.cancelled {
		return "", errCancelled
	}
	return m.input.Value(), nil
}
