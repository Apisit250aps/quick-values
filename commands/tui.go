package commands

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"maps"
	"qv/libs"
	"qv/utils"
	"strings"
	"unicode"
)

var (
	accent   = lipgloss.NewStyle().Foreground(lipgloss.Color("#B9A0FF")).Bold(true)
	muted    = lipgloss.NewStyle().Foreground(lipgloss.Color("#9295A5"))
	selected = lipgloss.NewStyle().Foreground(lipgloss.Color("#161322")).Background(lipgloss.Color("#B9A0FF")).Bold(true)
	panel    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#51486A")).Padding(1, 2)
)

type copyResult struct{ err error }
type model struct {
	store                        libs.Store
	tokens                       map[string]string
	cursor, width, height, field int
	mode, status, editing        string
	reveal                       bool
	search, key, value           textinput.Model
}

func newModel(store libs.Store, tokens map[string]string) model {
	search, key, value := textinput.New(), textinput.New(), textinput.New()
	search.Placeholder = "Search keys…"
	key.Placeholder = "e.g. github_token"
	value.Placeholder = "Secret value"
	value.EchoMode = textinput.EchoPassword
	value.EchoCharacter = '•'
	for _, input := range []*textinput.Model{&search, &key, &value} {
		input.SetVirtualCursor(true)
		input.SetWidth(36)
	}
	return model{store: store, tokens: tokens, width: 80, height: 24, search: search, key: key, value: value, status: "Values are hidden · stored locally in ~/.qv"}
}

func (m model) Init() tea.Cmd { return nil }
func (m model) keys() []string {
	keys := []string{}
	for _, key := range libs.Keys(m.tokens) {
		if strings.Contains(strings.ToLower(key), strings.ToLower(m.search.Value())) {
			keys = append(keys, key)
		}
	}
	return keys
}
func (m model) chosen() string {
	keys := m.keys()
	if len(keys) == 0 {
		return ""
	}
	return keys[min(m.cursor, len(keys)-1)]
}
func (m *model) save(tokens map[string]string, status string) {
	if err := m.store.Save(tokens); err != nil {
		m.status = "Could not save: " + err.Error()
		return
	}
	m.tokens, m.status, m.mode, m.reveal = tokens, status, "", false
	m.key.Reset()
	m.value.Reset()
	m.cursor = max(0, min(m.cursor, len(m.keys())-1))
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		for _, input := range []*textinput.Model{&m.search, &m.key, &m.value} {
			input.SetWidth(max(4, min(60, m.width-16)))
		}
	case copyResult:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = "Copied to clipboard"
		}
	case tea.KeyPressMsg:
		k := msg.String()
		if k == "ctrl+c" {
			return m, tea.Quit
		}
		if k == "esc" {
			if m.mode == "" {
				m.search.Reset()
			}
			m.mode = ""
			m.search.Blur()
			m.key.Reset()
			m.value.Reset()
			m.reveal = false
			return m, nil
		}
		if m.mode == "form" {
			if k == "tab" || k == "shift+tab" || k == "enter" && m.field == 0 {
				if m.editing != "" {
					m.field = 1
				} else {
					m.field = 1 - m.field
				}
				m.key.Blur()
				m.value.Blur()
				if m.field == 0 {
					return m, m.key.Focus()
				}
				return m, m.value.Focus()
			}
			if k == "enter" {
				key := m.key.Value()
				if strings.TrimSpace(key) == "" {
					m.status = "Enter a key first"
					return m, nil
				}
				if _, exists := m.tokens[key]; exists && m.editing == "" {
					m.status = "Key already exists · use e to edit"
					return m, nil
				}
				tokens := maps.Clone(m.tokens)
				tokens[key] = m.value.Value()
				m.save(tokens, "Saved "+safe(key))
				return m, nil
			}
		} else if m.mode == "delete" {
			if k == "y" {
				tokens := maps.Clone(m.tokens)
				delete(tokens, m.chosen())
				m.save(tokens, "Value deleted")
			}
			if k == "n" {
				m.mode = ""
			}
			return m, nil
		} else if m.mode == "search" {
			if k == "enter" {
				m.mode = ""
				m.search.Blur()
				return m, nil
			}
		} else {
			switch k {
			case "q":
				return m, tea.Quit
			case "up", "k":
				m.cursor = max(0, m.cursor-1)
				m.reveal = false
			case "down", "j":
				m.cursor = min(max(0, len(m.keys())-1), m.cursor+1)
				m.reveal = false
			case "/":
				m.mode = "search"
				m.reveal = false
				return m, m.search.Focus()
			case "a", "e":
				if k == "e" && len(m.keys()) == 0 {
					return m, nil
				}
				m.mode, m.editing, m.field, m.reveal = "form", "", 0, false
				m.key.Reset()
				m.value.Reset()
				m.key.Blur()
				m.value.Blur()
				m.status = "Enter a key and value · values stay masked"
				if k == "e" {
					m.editing = m.chosen()
					m.key.SetValue(m.editing)
					m.value.SetValue(m.tokens[m.editing])
					m.field = 1
					return m, m.value.Focus()
				}
				return m, m.key.Focus()
			case "d":
				if len(m.keys()) > 0 {
					m.mode = "delete"
					m.reveal = false
				}
			case "r":
				m.reveal = !m.reveal
			case "enter", "c":
				if len(m.keys()) > 0 {
					value := m.tokens[m.chosen()]
					m.status = "Copying…"
					return m, func() tea.Msg { return copyResult{utils.Copy(value)} }
				}
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	if m.mode == "search" {
		m.search, cmd = m.search.Update(msg)
		m.cursor = 0
	}
	if m.mode == "form" {
		if m.field == 0 {
			m.key, cmd = m.key.Update(msg)
		} else {
			m.value, cmd = m.value.Update(msg)
		}
	}
	return m, cmd
}

// safe prevents stored control characters from becoming terminal commands.
func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
func (m model) View() tea.View {
	width := max(8, min(76, m.width-8))
	var body strings.Builder
	body.WriteString(accent.Render("QUICK VAULT") + "  " + muted.Render("Your values, a keystroke away") + "\n\n")
	help := "↑/↓ move · / search · enter copy · a add · e edit\nr reveal · d delete · esc clear · q quit"
	if m.mode == "form" {
		title := "Add a value"
		if m.editing != "" {
			title = "Edit value"
		}
		body.WriteString(accent.Render(title) + "\n\nKey\n" + m.key.View() + "\n\nValue\n" + m.value.View() + "\n")
		help = "tab next field · enter continue/save · esc cancel"
	} else {
		body.WriteString(m.search.View() + "\n\n")
		keys := m.keys()
		rows := max(1, m.height-17)
		start := max(0, m.cursor-rows+1)
		if len(keys) == 0 {
			body.WriteString(muted.Render("No matching keys. Press a to add a value.") + "\n")
		}
		for i := start; i < min(len(keys), start+rows); i++ {
			line := "  " + ansi.Truncate(safe(keys[i]), max(1, width-4), "…")
			if i == m.cursor {
				line = selected.Width(width).Render("› " + ansi.Truncate(safe(keys[i]), max(1, width-4), "…"))
			}
			body.WriteString(line + "\n")
		}
		body.WriteString("\n" + muted.Render(fmt.Sprintf("%d / %d keys", len(keys), len(m.tokens))))
		if len(keys) > 0 {
			value := "••••••••"
			if m.reveal {
				value = safe(m.tokens[m.chosen()])
			}
			body.WriteString("\n" + ansi.Truncate(value, width, "…"))
		}
		if m.mode == "delete" {
			body.WriteString("\n\n" + accent.Render("Delete "+safe(m.chosen())+"?"))
			help = "y delete · n / esc cancel"
		}
		if m.mode == "search" {
			help = "type to filter · enter done · esc cancel"
		}
	}
	body.WriteString("\n\n" + muted.Render(ansi.Truncate(safe(m.status), width, "…")))
	content := panel.Width(width).Render(body.String()) + "\n" + muted.Render(help)
	if m.width < 50 || m.height < 20 {
		content = "Quick Vault\nResize terminal to at least 50×20.\nctrl+c quit"
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
