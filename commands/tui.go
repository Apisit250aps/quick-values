package commands

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"qv/libs"
	"qv/utils"
	"strings"
	"unicode"
)

type model struct {
	store                        libs.Store
	tokens                       map[string]string
	cursor, width, height, field int
	mode, status, editing        string
	reveal                       bool
	search, key, value, auth     textinput.Model
	pending                      vaultAction
	previous, revealed           string
	copyValue                    func(string) error
}

func newModel(store libs.Store, tokens map[string]string) model {
	names := map[string]string{}
	for name := range tokens {
		names[name] = ""
	}
	search, key, value := textinput.New(), textinput.New(), textinput.New()
	auth := textinput.New()
	auth.Placeholder = "Encryption password"
	auth.EchoMode = textinput.EchoPassword
	auth.EchoCharacter = '•'
	search.Placeholder = "Search keys…"
	key.Placeholder = "e.g. github_token"
	value.Placeholder = "Secret value"
	value.EchoMode = textinput.EchoPassword
	value.EchoCharacter = '•'
	for _, input := range []*textinput.Model{&search, &key, &value, &auth} {
		input.SetVirtualCursor(true)
		input.SetWidth(36)
		styleInput(input)
	}
	return model{store: store, tokens: names, width: 80, height: 24, search: search, key: key, value: value, auth: auth, copyValue: utils.Copy, status: "Copied values stay hidden"}
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
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		for _, input := range []*textinput.Model{&m.search, &m.key, &m.value, &m.auth} {
			input.SetWidth(max(4, min(60, m.width-16)))
		}
	case actionResult:
		m.auth.Reset()
		m.auth.Blur()
		if msg.err != nil {
			m.mode, m.status = m.previous, msg.err.Error()
			m.pending = vaultAction{}
			return m, nil
		}
		m.tokens, m.status, m.mode = msg.names, msg.status, ""
		m.revealed, m.reveal = msg.revealed, m.pending.kind == "reveal"
		m.pending = vaultAction{}
		m.key.Reset()
		m.value.Reset()
		m.cursor = max(0, min(m.cursor, len(m.keys())-1))
		return m, nil
	case tea.KeyPressMsg:
		k := msg.String()
		if k == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mode == "busy" {
			return m, nil
		}
		if m.mode == "auth" {
			if k == "esc" {
				m.mode, m.status = m.previous, "Action cancelled"
				m.auth.Reset()
				m.auth.Blur()
				m.pending = vaultAction{}
				return m, nil
			}
			if k == "enter" {
				password := m.auth.Value()
				m.auth.Reset()
				m.auth.Blur()
				cmd := m.startAction(password)
				return m, cmd
			}
			var cmd tea.Cmd
			m.auth, cmd = m.auth.Update(msg)
			return m, cmd
		}
		if k == "esc" {
			if m.mode == "" {
				m.search.Reset()
			}
			m.mode = ""
			m.search.Blur()
			m.key.Reset()
			m.value.Reset()
			m.reveal, m.revealed = false, ""
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
				cmd := m.requestAction(vaultAction{kind: "save", name: key, value: m.value.Value(), replacing: m.editing != ""})
				return m, cmd
			}
		} else if m.mode == "delete" {
			if k == "y" {
				cmd := m.requestAction(vaultAction{kind: "delete", name: m.chosen()})
				return m, cmd
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
				m.reveal, m.revealed = false, ""
			case "down", "j":
				m.cursor = min(max(0, len(m.keys())-1), m.cursor+1)
				m.reveal, m.revealed = false, ""
			case "/":
				m.mode = "search"
				m.reveal, m.revealed = false, ""
				return m, m.search.Focus()
			case "a", "e":
				if k == "e" && len(m.keys()) == 0 {
					return m, nil
				}
				m.mode, m.editing, m.field, m.reveal = "form", "", 0, false
				m.revealed = ""
				m.key.Reset()
				m.value.Reset()
				m.key.Blur()
				m.value.Blur()
				m.status = "Enter a key and value · values stay masked"
				if k == "e" {
					m.editing = m.chosen()
					m.key.SetValue(m.editing)
					m.value.Placeholder = "Enter replacement value"
					m.field = 1
					return m, m.value.Focus()
				}
				return m, m.key.Focus()
			case "d":
				if len(m.keys()) > 0 {
					m.mode = "delete"
					m.reveal, m.revealed = false, ""
				}
			case "r":
				if m.reveal {
					m.reveal, m.revealed = false, ""
					return m, nil
				}
				if len(m.keys()) > 0 {
					cmd := m.requestAction(vaultAction{kind: "reveal", name: m.chosen()})
					return m, cmd
				}
			case "enter", "c":
				if len(m.keys()) > 0 {
					cmd := m.requestAction(vaultAction{kind: "copy", name: m.chosen()})
					return m, cmd
				}
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	if m.mode == "auth" {
		m.auth, cmd = m.auth.Update(msg)
	}
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
	storage := peach.Render(m.store.StorageLabel())
	if m.store.Encrypted() {
		storage = mint.Render(m.store.StorageLabel())
	}
	body.WriteString(accent.Render("QUICK VAULT") + "  " + muted.Render("Your values, a keystroke away") + "\n" + storage + "\n\n")
	help := "↑/↓ move · / search · enter copy · a add · e edit\nr reveal · d delete · esc clear · q quit"
	if m.mode == "auth" {
		body.WriteString(accent.Render("Authorize "+m.pending.kind) + "\n\n" + sky.Render(safe(m.pending.name)) + "\n\n" + m.auth.View() + "\n")
		help = "enter authorize this action · esc cancel"
	} else if m.mode == "busy" {
		body.WriteString(mint.Render("Working…") + "\n")
		help = "Please wait for the action to finish"
	} else if m.mode == "form" {
		title := "Add a value"
		if m.editing != "" {
			title = "Edit value"
		}
		body.WriteString(accent.Render(title) + "\n\n" + sky.Render("Key") + "\n" + m.key.View() + "\n\n" + sky.Render("Value") + "\n" + m.value.View() + "\n")
		help = "tab next field · enter continue/save · esc cancel"
	} else {
		body.WriteString(m.search.View() + "\n\n")
		keys := m.keys()
		rows := max(1, m.height-19)
		start := max(0, m.cursor-rows+1)
		if len(keys) == 0 {
			body.WriteString(muted.Render("No matching keys. Press a to add a value.") + "\n")
		}
		for i := start; i < min(len(keys), start+rows); i++ {
			line := sky.Render("  " + ansi.Truncate(safe(keys[i]), max(1, width-4), "…"))
			if i == m.cursor {
				line = selected.Width(width).Render("› " + ansi.Truncate(safe(keys[i]), max(1, width-4), "…"))
			}
			body.WriteString(line + "\n")
		}
		body.WriteString("\n" + muted.Render(fmt.Sprintf("%d / %d keys", len(keys), len(m.tokens))))
		if len(keys) > 0 {
			value := "••••••••"
			if m.reveal {
				value = safe(m.revealed)
			}
			body.WriteString("\n" + peach.Render(ansi.Truncate(value, width, "…")))
		}
		if m.mode == "delete" {
			body.WriteString("\n\n" + rose.Render("Delete "+safe(m.chosen())+"?"))
			help = "y delete · n / esc cancel"
		}
		if m.mode == "search" {
			help = "type to filter · enter done · esc cancel"
		}
	}
	body.WriteString("\n\n" + peach.Render(ansi.Truncate(safe(m.status), width, "…")))
	content := panel.Width(width).Render(body.String()) + "\n" + sky.Render(help)
	if m.width < 50 || m.height < 22 {
		content = accent.Render("Quick Vault") + "\n" + peach.Render("Resize terminal to at least 50×22.") + "\n" + sky.Render("ctrl+c quit")
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
