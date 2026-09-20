package commands

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"qv/libs"
)

type vaultAction struct {
	kind, name, value string
	replacing         bool
}
type actionResult struct {
	names            map[string]string
	revealed, status string
	err              error
}

// Each action opens and closes its own authenticated session. Browsing has no key.
func executeAction(path, password string, action vaultAction, copyValue func(string) error) actionResult {
	store := libs.Store{Path: path}
	defer store.Close()
	values, err := store.Unlock(password)
	if err != nil {
		return actionResult{err: err}
	}
	defer clear(values)
	result := actionResult{}
	value, exists := values[action.name]
	switch action.kind {
	case "copy", "reveal":
		if !exists {
			return actionResult{err: fmt.Errorf("key no longer exists; reopen the vault")}
		}
		if action.kind == "copy" {
			if err := copyValue(value); err != nil {
				return actionResult{err: err}
			}
			result.status = "Copied to clipboard · value hidden"
		} else {
			result.revealed = value
			result.status = "Value revealed · r hides it"
		}
	case "save":
		if exists && !action.replacing {
			return actionResult{err: fmt.Errorf("key already exists; reopen the vault to edit it")}
		}
		if !exists && action.replacing {
			return actionResult{err: fmt.Errorf("key no longer exists; reopen the vault")}
		}
		values[action.name] = action.value
		if err := store.Save(values); err != nil {
			return actionResult{err: err}
		}
		result.status = "Saved " + safe(action.name)
	case "delete":
		if !exists {
			return actionResult{err: fmt.Errorf("key no longer exists; reopen the vault")}
		}
		delete(values, action.name)
		if err := store.Save(values); err != nil {
			return actionResult{err: err}
		}
		result.status = "Value deleted"
	}
	result.names, err = store.Browse()
	if err != nil {
		result.err = err
	}
	return result
}

func (m *model) requestAction(action vaultAction) tea.Cmd {
	m.pending, m.previous = action, m.mode
	m.reveal, m.revealed = false, ""
	if m.store.Encrypted() {
		m.mode, m.status = "auth", "Password required for this action"
		m.auth.Reset()
		return m.auth.Focus()
	}
	return m.startAction("")
}
func (m *model) startAction(password string) tea.Cmd {
	m.mode, m.status = "busy", "Working…"
	path, action, copyValue := m.store.Path, m.pending, m.copyValue
	return func() tea.Msg { return executeAction(path, password, action, copyValue) }
}
