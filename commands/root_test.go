package commands

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"qv/libs"
	"strings"
	"testing"
)

func testRunner(t *testing.T, answers ...string) (runner, *bytes.Buffer) {
	t.Helper()
	out := new(bytes.Buffer)
	store := &libs.Store{Path: filepath.Join(t.TempDir(), ".qv")}
	t.Cleanup(store.Close)
	r := runner{store: store, out: out, copy: func(string) error { return nil }}
	r.prompt = func(_, _ string, _ bool) (string, error) {
		if len(answers) == 0 {
			return "", errCancelled
		}
		a := answers[0]
		answers = answers[1:]
		return a, nil
	}
	r.interactive = func(libs.Store, map[string]string) error { return nil }
	return r, out
}
func TestInitializationGate(t *testing.T) {
	r, out := testRunner(t)
	for _, args := range [][]string{nil, {"list"}, {"get", "api"}, {"set", "api", "value"}, {"del", "api"}, {"change-password"}} {
		if !errors.Is(r.run(args), libs.ErrNotInitialized) {
			t.Fatalf("not gated: %v", args)
		}
	}
	if err := r.run([]string{"help"}); err != nil || !strings.Contains(out.String(), "qv init") {
		t.Fatal("help unavailable", err)
	}
}
func TestCopyPrivacyAndShow(t *testing.T) {
	r, out := testRunner(t, "")
	if err := r.run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	if err := r.run([]string{"set", "api", "private-secret"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "private-secret") {
		t.Fatal("set leaked secret")
	}
	copied := ""
	r.copy = func(value string) error { copied = value; return nil }
	out.Reset()
	if err := r.run([]string{"get", "api"}); err != nil {
		t.Fatal(err)
	}
	if copied != "private-secret" || strings.Contains(out.String(), "private-secret") {
		t.Fatal("get privacy failed")
	}
	out.Reset()
	if err := r.run([]string{"get", "api", "--show"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "private-secret") {
		t.Fatal("show missing")
	}
	r.copy = func(string) error { return errors.New("clipboard unavailable") }
	out.Reset()
	if err := r.run([]string{"get", "api", "--show"}); err == nil {
		t.Fatal("clipboard failure hidden")
	}
	if strings.Contains(out.String(), "Copied") || strings.Contains(out.String(), "private-secret") {
		t.Fatal("failed copy claims success or leaks value")
	}
	called := false
	r.interactive = func(_ libs.Store, values map[string]string) error {
		_, called = values["api"]
		if values["api"] != "" {
			t.Fatal("browser received plaintext value")
		}
		return nil
	}
	if err := r.run([]string{"list"}); err != nil || !called {
		t.Fatal("list is not interactive", err)
	}
}
func TestSetupCancellationPreservesData(t *testing.T) {
	for _, answers := range [][]string{{"no"}, {"RESET"}, {"RESET", "one", "different"}} {
		r, _ := testRunner(t, answers...)
		if err := r.store.Initialize("", false); err != nil {
			t.Fatal(err)
		}
		if err := r.store.Save(map[string]string{"keep": "value"}); err != nil {
			t.Fatal(err)
		}
		before, _ := os.ReadFile(r.store.Path)
		if err := r.run([]string{"reset"}); err == nil {
			t.Fatal("expected cancellation or mismatch")
		}
		after, _ := os.ReadFile(r.store.Path)
		if !bytes.Equal(before, after) {
			t.Fatal("cancelled reset changed data")
		}
	}
}
func TestEncryptedCommandFlow(t *testing.T) {
	r, out := testRunner(t, "old", "old", "old", "old", "new", "new", "new", "RESET", "")
	for _, args := range [][]string{{"init"}, {"set", "api", "secret-value"}, {"change-password"}, {"get", "api"}, {"reset"}} {
		if err := r.run(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if strings.Contains(out.String(), "secret-value") {
		t.Fatal("secret leaked")
	}
	status, err := r.store.Status()
	if err != nil || status.Encrypted || !status.Initialized {
		t.Fatal("reset status", err)
	}
	values, err := r.store.Load()
	if err != nil || len(values) != 0 {
		t.Fatal("reset values", err)
	}
}
func TestCopyHidesRevealedTUIValue(t *testing.T) {
	m := newModel(libs.Store{}, map[string]string{"api": "private-secret"})
	m.reveal = true
	m = press(m, 'c')
	if strings.Contains(m.View().Content, "private-secret") {
		t.Fatal("copy left value visible")
	}
}

func TestEncryptedBrowsingDoesNotPrompt(t *testing.T) {
	r, _ := testRunner(t)
	if err := r.store.Initialize("password", false); err != nil {
		t.Fatal(err)
	}
	if err := r.store.Save(map[string]string{"api": "secret"}); err != nil {
		t.Fatal(err)
	}
	prompts := 0
	r.prompt = func(_, _ string, _ bool) (string, error) { prompts++; return "password", nil }
	visits := 0
	r.interactive = func(_ libs.Store, names map[string]string) error {
		visits++
		if len(names) != 1 || names["api"] != "" {
			t.Fatal("browse leaked values")
		}
		return nil
	}
	for _, args := range [][]string{nil, {"list"}, {"get"}} {
		if err := r.run(args); err != nil {
			t.Fatal(err)
		}
	}
	if prompts != 0 || visits != 3 {
		t.Fatal("browsing prompted", prompts, visits)
	}
	for _, args := range [][]string{{"get", "api"}, {"get", "api", "--show"}, {"set", "api", "updated"}, {"del", "api"}} {
		before := prompts
		if err := r.run(args); err != nil {
			t.Fatal(err)
		}
		if prompts != before+1 {
			t.Fatal("action did not ask again", args)
		}
	}
}
