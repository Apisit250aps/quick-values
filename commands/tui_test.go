package commands

import (
	tea "charm.land/bubbletea/v2"
	"os"
	"path/filepath"
	"qv/libs"
	"strings"
	"testing"
)

func press(m model, code rune) model {
	next, cmd := m.Update(tea.KeyPressMsg{Code: code})
	result := next.(model)
	if result.mode == "busy" && cmd != nil {
		next, _ = result.Update(cmd())
		result = next.(model)
	}
	return result
}

func TestVaultWorkflow(t *testing.T) {
	store := libs.Store{Path: filepath.Join(t.TempDir(), ".qv")}
	if err := store.Initialize("", false); err != nil {
		t.Fatal(err)
	}
	m := newModel(store, map[string]string{})
	m = press(m, 'a')
	m.key.SetValue("api")
	m = press(m, tea.KeyEnter)
	m.value.SetValue("private-value")
	m = press(m, tea.KeyEnter)
	if m.mode != "" || len(m.tokens) != 1 || m.tokens["api"] != "" {
		t.Fatal("add failed")
	}
	if strings.Contains(m.View().Content, "private-value") {
		t.Fatal("secret visible by default")
	}
	m = press(m, 'r')
	if !strings.Contains(m.View().Content, "private-value") {
		t.Fatal("reveal failed")
	}
	m = press(m, 'e')
	m.value.SetValue("updated-value")
	m = press(m, tea.KeyEnter)
	saved, err := store.Load()
	if err != nil || saved["api"] != "updated-value" {
		t.Fatal("edit was not persisted", err)
	}
	m = press(m, 'd')
	m = press(m, 'n')
	if _, ok := m.tokens["api"]; !ok {
		t.Fatal("cancel deleted key")
	}
	m = press(m, 'd')
	m = press(m, 'y')
	saved, err = store.Load()
	if err != nil || len(saved) != 0 {
		t.Fatal("delete failed", err)
	}
}

func TestFailedSavePreservesState(t *testing.T) {
	m := newModel(libs.Store{Path: filepath.Join(t.TempDir(), "missing", ".qv")}, map[string]string{"api": "old"})
	m = press(m, 'e')
	m.value.SetValue("new")
	m = press(m, tea.KeyEnter)
	if m.tokens["api"] != "" || m.mode != "form" {
		t.Fatal("failed save changed state or discarded form")
	}
}

func TestInvalidVaultIsNotOverwritten(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".qv")
	original := []byte(`{"broken":`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"set", "key", "value"}, os.Stdout); err == nil {
		t.Fatal("expected load error")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(original) {
		t.Fatal("invalid vault overwritten")
	}
}

func TestTUIRequiresPasswordForEveryAction(t *testing.T) {
	store := libs.Store{Path: filepath.Join(t.TempDir(), ".qv")}
	if err := store.Initialize("password", false); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(map[string]string{"api": "secret"}); err != nil {
		t.Fatal(err)
	}
	names, err := store.Browse()
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(store, names)
	copies := 0
	m.copyValue = func(value string) error {
		if value != "secret" {
			t.Fatal("wrong copy")
		}
		copies++
		return nil
	}
	m = press(m, 'c')
	if m.mode != "auth" || copies != 0 {
		t.Fatal("copy bypassed authorization")
	}
	m.auth.SetValue("wrong")
	m = press(m, tea.KeyEnter)
	if copies != 0 || strings.Contains(m.View().Content, "secret") {
		t.Fatal("wrong password exposed value")
	}
	m = press(m, 'c')
	m.auth.SetValue("password")
	m = press(m, tea.KeyEnter)
	if copies != 1 || m.mode != "" || m.tokens["api"] != "" || m.auth.Value() != "" {
		t.Fatal("copy/session cleanup failed")
	}
	m = press(m, 'c')
	if m.mode != "auth" || copies != 1 {
		t.Fatal("second copy reused password")
	}
	m = press(m, tea.KeyEscape)
	m = press(m, 'r')
	if m.mode != "auth" {
		t.Fatal("reveal bypassed password")
	}
	m.auth.SetValue("password")
	m = press(m, tea.KeyEnter)
	if !strings.Contains(m.View().Content, "secret") {
		t.Fatal("reveal failed")
	}
	m = press(m, 'r')
	if m.revealed != "" {
		t.Fatal("hide retained value")
	}
	m = press(m, 'e')
	if m.mode != "form" || m.value.Value() != "" {
		t.Fatal("edit decrypted without password")
	}
	m.value.SetValue("replacement")
	m = press(m, tea.KeyEnter)
	if m.mode != "auth" {
		t.Fatal("save bypassed password")
	}
	m = press(m, tea.KeyEscape)
	if m.mode != "form" || m.value.Value() != "replacement" {
		t.Fatal("cancel lost draft")
	}
	m = press(m, tea.KeyEnter)
	m.auth.SetValue("password")
	m = press(m, tea.KeyEnter)
	saved, err := store.Unlock("password")
	if err != nil || saved["api"] != "replacement" {
		t.Fatal("authorized save failed", err)
	}
	m = press(m, 'd')
	m = press(m, 'y')
	if m.mode != "auth" {
		t.Fatal("delete bypassed password")
	}
}
