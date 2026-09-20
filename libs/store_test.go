package libs

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func vault(t *testing.T) *Store {
	t.Helper()
	s := &Store{Path: filepath.Join(t.TempDir(), ".qv")}
	t.Cleanup(s.Close)
	return s
}
func contents(t *testing.T, s *Store) []byte {
	t.Helper()
	b, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestInitializationAndMigration(t *testing.T) {
	for _, password := range []string{"", "correct password"} {
		t.Run(map[bool]string{true: "encrypted", false: "plain"}[password != ""], func(t *testing.T) {
			s := vault(t)
			if _, err := s.Load(); err != ErrNotInitialized {
				t.Fatalf("uninitialized: %v", err)
			}
			if err := os.WriteFile(s.Path, []byte(`{"version":"legacy-key","token":"legacy-secret"}`), 0644); err != nil {
				t.Fatal(err)
			}
			status, err := s.Status()
			if err != nil || !status.Legacy || status.Initialized {
				t.Fatal("legacy status", err)
			}
			if err := s.Initialize(password, false); err != nil {
				t.Fatal(err)
			}
			status, err = s.Status()
			if err != nil || !status.Initialized || status.Encrypted != (password != "") {
				t.Fatal("status", err)
			}
			values, err := s.Unlock(password)
			if err != nil || values["token"] != "legacy-secret" || values["version"] != "legacy-key" {
				t.Fatal("migration", err)
			}
			if password != "" && bytes.Contains(contents(t, s), []byte("legacy-secret")) {
				t.Fatal("plaintext leaked")
			}
			info, _ := os.Stat(s.Path)
			if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
				t.Fatal("permissions", info.Mode())
			}
			before := contents(t, s)
			if err := s.Initialize("another", false); err == nil {
				t.Fatal("reinitialization allowed")
			}
			if !bytes.Equal(before, contents(t, s)) {
				t.Fatal("reinitialization changed file")
			}
		})
	}
}

func TestEncryptionAndPasswordRotation(t *testing.T) {
	s := vault(t)
	if err := s.Initialize("old-password", false); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{"secret-key": "secret-value"}
	if err := s.Save(values); err != nil {
		t.Fatal(err)
	}
	before := contents(t, s)
	if err := s.Save(values); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, contents(t, s)) {
		t.Fatal("nonce reused")
	}
	before = contents(t, s)
	if err := s.ChangePassword("wrong", "new-password"); err == nil {
		t.Fatal("wrong password accepted")
	}
	if !bytes.Equal(before, contents(t, s)) {
		t.Fatal("wrong password changed vault")
	}
	if err := s.ChangePassword("old-password", "new-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Unlock("old-password"); err == nil {
		t.Fatal("old password still works")
	}
	got, err := s.Unlock("new-password")
	if err != nil || got["secret-key"] != "secret-value" {
		t.Fatal("rotation lost values", err)
	}
	if bytes.Contains(contents(t, s), []byte("secret-value")) {
		t.Fatal("plaintext leaked")
	}
	// Corrupt authenticated ciphertext while preserving valid JSON and metadata.
	e, err := parse(contents(t, s))
	if err != nil {
		t.Fatal(err)
	}
	e.Entries["secret-key"][len(e.Entries["secret-key"])-1] ^= 1
	damaged, _ := json.Marshal(e)
	damaged = append([]byte(fileHeader), damaged...)
	if err := os.WriteFile(s.Path, damaged, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Unlock("new-password"); err == nil {
		t.Fatal("tampering accepted")
	}
}

func TestResetAndStaleSession(t *testing.T) {
	s := vault(t)
	if err := s.Initialize("", false); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(map[string]string{"old": "value"}); err != nil {
		t.Fatal(err)
	}
	stale := &Store{Path: s.Path}
	if _, err := stale.Unlock(""); err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize("new", true); err != nil {
		t.Fatal(err)
	}
	if err := stale.Save(map[string]string{"stale": "value"}); err == nil {
		t.Fatal("stale writer overwrote vault")
	}
	values, err := s.Unlock("new")
	if err != nil || len(values) != 0 {
		t.Fatal("reset retained data", err)
	}
	if err := os.WriteFile(s.Path, []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Initialize("", true); err != nil {
		t.Fatal("reset damaged file", err)
	}
	values, err = s.Load()
	if err != nil || len(values) != 0 {
		t.Fatal("reset failed", err)
	}
}

func TestPlaintextUpgradeAndFailedWrite(t *testing.T) {
	s := vault(t)
	if err := s.Initialize("", false); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(map[string]string{"api": "private"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ChangePassword("", "password"); err != nil {
		t.Fatal(err)
	}
	values, err := s.Unlock("password")
	if err != nil || values["api"] != "private" {
		t.Fatal("upgrade failed", err)
	}
	before := contents(t, s)
	if err := os.WriteFile(s.Path+".lock", nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.ChangePassword("password", "replacement"); err == nil {
		t.Fatal("locked write accepted")
	}
	if !bytes.Equal(before, contents(t, s)) {
		t.Fatal("failed rotation changed data")
	}
}

func TestBrowseNeverUnlocks(t *testing.T) {
	s := vault(t)
	if err := s.Initialize("password", false); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(map[string]string{"visible-name": "hidden-value"}); err != nil {
		t.Fatal(err)
	}
	data := contents(t, s)
	if !bytes.Contains(data, []byte("visible-name")) || bytes.Contains(data, []byte("hidden-value")) {
		t.Fatal("value-only format failed")
	}
	names, err := s.Browse()
	if err != nil || len(names) != 1 || names["visible-name"] != "" || len(s.key) != 0 {
		t.Fatal("browse requires or retains a key", err)
	}
	if err := s.Save(map[string]string{"bypass": "value"}); err == nil {
		t.Fatal("browse permitted unauthenticated write")
	}
	if _, err := s.Unlock("wrong"); err == nil {
		t.Fatal("wrong password accepted")
	}
}

func TestWholeVaultMigration(t *testing.T) {
	s := vault(t)
	salt := []byte("0123456789abcdef")
	key := derive("password", salt)
	defer clear(key)
	gcm, err := aead(key)
	if err != nil {
		t.Fatal(err)
	}
	old := envelope{Mode: "argon2id-aes256gcm", Salt: salt, Data: gcm.Seal(nil, nil, []byte(`{"api":"secret"}`), aad(salt))}
	encoded, _ := json.Marshal(old)
	if err := os.WriteFile(s.Path, append([]byte(fileHeader), encoded...), 0600); err != nil {
		t.Fatal(err)
	}
	before := contents(t, s)
	if _, err := s.Browse(); err != ErrMigrationRequired {
		t.Fatal("missing migration gate", err)
	}
	if err := s.Migrate("wrong"); err == nil {
		t.Fatal("wrong migration password accepted")
	}
	if !bytes.Equal(before, contents(t, s)) {
		t.Fatal("failed migration changed data")
	}
	if err := s.Migrate("password"); err != nil {
		t.Fatal(err)
	}
	names, err := s.Browse()
	if err != nil || len(names) != 1 || names["api"] != "" {
		t.Fatal("migrated browsing failed", err)
	}
	values, err := s.Unlock("password")
	if err != nil || values["api"] != "secret" {
		t.Fatal("migration lost value", err)
	}
}

func TestEncryptedEmptyVaultAndEntryBinding(t *testing.T) {
	s := vault(t)
	if err := s.Initialize("password", false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Unlock("wrong"); err == nil {
		t.Fatal("empty vault accepted wrong password")
	}
	if err := s.Save(map[string]string{"a": "first", "b": "second"}); err != nil {
		t.Fatal(err)
	}
	original := contents(t, s)
	for _, kind := range []string{"swap", "delete"} {
		e, err := parse(original)
		if err != nil {
			t.Fatal(err)
		}
		if kind == "swap" {
			e.Entries["a"], e.Entries["b"] = e.Entries["b"], e.Entries["a"]
		} else {
			delete(e.Entries, "a")
		}
		data, _ := json.Marshal(e)
		if err := os.WriteFile(s.Path, append([]byte(fileHeader), data...), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Unlock("password"); err == nil {
			t.Fatalf("%s tampering accepted", kind)
		}
	}
}
