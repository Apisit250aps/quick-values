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
	next, _ := m.Update(tea.KeyPressMsg{Code: code})
	return next.(model)
}

func TestVaultWorkflow(t *testing.T) {
	store := libs.Store{Path: filepath.Join(t.TempDir(), ".qv")}
	m := newModel(store, map[string]string{})
	m = press(m, 'a')
	m.key.SetValue("api")
	m = press(m, tea.KeyEnter)
	m.value.SetValue("private-value")
	m = press(m, tea.KeyEnter)
	if m.mode != "" || m.tokens["api"] != "private-value" {
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
	if m.tokens["api"] != "old" || m.mode != "form" {
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
