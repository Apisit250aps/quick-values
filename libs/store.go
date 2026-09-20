// Package libs implements versioned, optionally encrypted vault persistence.
package libs

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/crypto/argon2"
)

const fileHeader = "QV1\n"

var ErrNotInitialized = errors.New("vault is not initialized; run qv init first")
var ErrMigrationRequired = errors.New("old vault encrypts key names; unlock once to migrate to value-only encryption")

const valuesMode = "argon2id-aes256gcm-values"

// Store holds an unlocked vault session. Passwords are never persisted.
type Store struct {
	Path                   string
	key, salt, snapshot    []byte
	initialized, encrypted bool
}

type envelope struct {
	Mode    string            `json:"mode"`
	Salt    []byte            `json:"salt,omitempty"`
	Data    []byte            `json:"data,omitempty"`
	Values  map[string]string `json:"values,omitempty"`
	Entries map[string][]byte `json:"encrypted_values,omitempty"`
	Verify  []byte            `json:"verify,omitempty"`
}

type Status struct{ Initialized, Encrypted, Legacy, NeedsMigration bool }

func DefaultStore() (Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Store{}, err
	}
	return Store{Path: filepath.Join(home, ".qv")}, nil
}

func (s Store) read() ([]byte, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read vault: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("vault is empty or damaged")
	}
	return data, nil
}

func parse(data []byte) (envelope, error) {
	var e envelope
	if !bytes.HasPrefix(data, []byte(fileHeader)) {
		return e, ErrNotInitialized
	}
	if err := json.Unmarshal(data[len(fileHeader):], &e); err != nil {
		return e, errors.New("invalid vault format")
	}
	switch e.Mode {
	case "plain":
		if len(e.Salt) != 0 || len(e.Data) != 0 || e.Entries != nil || len(e.Verify) != 0 {
			return e, errors.New("invalid plaintext vault")
		}
	case "argon2id-aes256gcm":
		if len(e.Salt) != 16 || len(e.Data) < 28 || e.Values != nil || e.Entries != nil || len(e.Verify) != 0 {
			return e, errors.New("invalid encrypted vault")
		}
	case valuesMode:
		if len(e.Salt) != 16 || len(e.Data) != 0 || e.Values != nil || len(e.Verify) < 28 {
			return e, errors.New("invalid value-encrypted vault")
		}
		for _, value := range e.Entries {
			if len(value) < 28 {
				return e, errors.New("invalid encrypted value")
			}
		}
	default:
		return e, errors.New("unsupported vault format")
	}
	return e, nil
}

func (s Store) Status() (Status, error) {
	data, err := s.read()
	if err != nil {
		return Status{}, err
	}
	if data == nil {
		return Status{}, nil
	}
	if !bytes.HasPrefix(data, []byte(fileHeader)) {
		if _, err := decodeValues(data); err != nil {
			return Status{}, err
		}
		return Status{Legacy: true}, nil
	}
	e, err := parse(data)
	return Status{Initialized: err == nil, Encrypted: e.Mode != "plain", NeedsMigration: e.Mode == "argon2id-aes256gcm"}, err
}

func decodeValues(data []byte) (map[string]string, error) {
	var values map[string]string
	if err := json.Unmarshal(data, &values); err != nil || values == nil {
		return nil, errors.New("invalid vault JSON; original file has not been changed")
	}
	return values, nil
}

// V1 fixes Argon2id parameters to prevent hostile files requesting arbitrary resources.
func derive(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
}
func aead(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCMWithRandomNonce(block)
}
func aad(salt []byte) []byte { return append([]byte(fileHeader+"argon2id-aes256gcm"), salt...) }

func (s *Store) Unlock(password string) (map[string]string, error) {
	data, err := s.read()
	if err != nil {
		return nil, err
	}
	e, err := parse(data)
	if err != nil {
		return nil, err
	}
	key := []byte(nil)
	if e.Mode != "plain" {
		if password == "" {
			return nil, errors.New("password is required")
		}
		key = derive(password, e.Salt)
	}
	values, err := decrypt(e, key)
	if err != nil {
		clear(key)
		return nil, err
	}
	s.Close()
	s.key, s.salt, s.snapshot = key, e.Salt, data
	s.initialized, s.encrypted = true, e.Mode != "plain"
	return values, nil
}

func decrypt(e envelope, key []byte) (map[string]string, error) {
	if e.Mode == "plain" {
		if e.Values == nil {
			return map[string]string{}, nil
		}
		return e.Values, nil
	}
	gcm, err := aead(key)
	if err != nil {
		return nil, errors.New("password is required")
	}
	if e.Mode == "argon2id-aes256gcm" {
		plain, err := gcm.Open(nil, nil, e.Data, aad(e.Salt))
		if err != nil {
			return nil, errors.New("incorrect password or damaged vault")
		}
		defer clear(plain)
		return decodeValues(plain)
	}
	if _, err := gcm.Open(nil, nil, e.Verify, manifestAAD(e)); err != nil {
		return nil, errors.New("incorrect password or damaged vault")
	}
	values := map[string]string{}
	for name, sealed := range e.Entries {
		plain, err := gcm.Open(nil, nil, sealed, valueAAD(e.Salt, name))
		if err != nil {
			return nil, errors.New("incorrect password or damaged value")
		}
		values[name] = string(plain)
		clear(plain)
	}
	return values, nil
}

// Browse returns names only. It never derives a key or decrypts values.
func (s *Store) Browse() (map[string]string, error) {
	data, err := s.read()
	if err != nil {
		return nil, err
	}
	e, err := parse(data)
	if err != nil {
		return nil, err
	}
	if e.Mode == "argon2id-aes256gcm" {
		return nil, ErrMigrationRequired
	}
	names := map[string]string{}
	for name := range e.Values {
		names[name] = ""
	}
	for name := range e.Entries {
		names[name] = ""
	}
	s.Close()
	s.encrypted = e.Mode != "plain"
	s.salt, s.snapshot = e.Salt, data
	return names, nil
}

// Migrate converts the old whole-vault format only after successful authentication.
func (s *Store) Migrate(password string) error {
	status, err := s.Status()
	if err != nil {
		return err
	}
	if !status.NeedsMigration {
		return nil
	}
	values, err := s.Unlock(password)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Save(values)
}

func valueAAD(salt []byte, name string) []byte {
	data, _ := json.Marshal([]any{fileHeader, valuesMode, "value", salt, name})
	return data
}
func manifestAAD(e envelope) []byte {
	names := make([]string, 0, len(e.Entries))
	for name := range e.Entries {
		names = append(names, name)
	}
	sort.Strings(names)
	data, _ := json.Marshal([]any{fileHeader, valuesMode, "manifest", e.Salt, names})
	return data
}

// Load supports an unlocked session, or opening a plaintext vault.
func (s *Store) Load() (map[string]string, error) {
	if !s.encrypted {
		return s.Unlock("")
	}
	data, err := s.read()
	if err != nil {
		return nil, err
	}
	e, err := parse(data)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(e.Salt, s.salt) {
		return nil, errors.New("vault changed; reopen it")
	}
	values, err := decrypt(e, s.key)
	if err == nil {
		s.snapshot = data
	}
	return values, err
}

func (s *Store) Close()         { clear(s.key); s.key = nil; s.initialized = false }
func (s Store) Encrypted() bool { return s.encrypted }
func (s Store) StorageLabel() string {
	if s.encrypted {
		return "VALUES ENCRYPTED · keys visible"
	}
	return "PLAIN TEXT · not encrypted"
}

// Initialize preserves legacy values. Reset replaces all values only after setup succeeds.
func (s *Store) Initialize(password string, reset bool) error {
	data, err := s.read()
	// Reset is also the recovery path for a damaged vault.
	if err != nil && !reset {
		return err
	}
	if err != nil {
		data, err = os.ReadFile(s.Path)
		if err != nil {
			return err
		}
	}
	values := map[string]string{}
	if !reset && data != nil {
		if bytes.HasPrefix(data, []byte(fileHeader)) {
			return errors.New("vault is already initialized; use qv change-password or qv reset")
		}
		values, err = decodeValues(data)
		if err != nil {
			return err
		}
	}
	return s.replace(password, values, data)
}

func (s *Store) ChangePassword(oldPassword, newPassword string) error {
	if newPassword == "" {
		return errors.New("new password must not be empty")
	}
	values, err := s.Unlock(oldPassword)
	if err != nil {
		return err
	}
	return s.replace(newPassword, values, s.snapshot)
}

func (s *Store) replace(password string, values map[string]string, snapshot []byte) error {
	next := Store{Path: s.Path, initialized: true, encrypted: password != "", snapshot: snapshot}
	if next.encrypted {
		next.salt = make([]byte, 16)
		if _, err := rand.Read(next.salt); err != nil {
			return err
		}
		next.key = derive(password, next.salt)
	}
	if err := next.Save(values); err != nil {
		next.Close()
		return err
	}
	s.Close()
	*s = next
	return nil
}

func (s *Store) Save(values map[string]string) error {
	if !s.initialized {
		return ErrNotInitialized
	}
	e := envelope{Mode: "plain", Values: values}
	if s.encrypted {
		gcm, err := aead(s.key)
		if err != nil {
			return err
		}
		e = envelope{Mode: valuesMode, Salt: s.salt, Entries: map[string][]byte{}}
		for name, value := range values {
			e.Entries[name] = gcm.Seal(nil, nil, []byte(value), valueAAD(s.salt, name))
		}
		e.Verify = gcm.Seal(nil, nil, nil, manifestAAD(e))
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	data = append([]byte(fileHeader), data...)
	// Serialize writers and reject stale sessions rather than overwriting newer data.
	lock, err := os.OpenFile(s.Path+".lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("vault is busy or cannot be locked: %w", err)
	}
	lock.Close()
	defer os.Remove(s.Path + ".lock")
	current, err := os.ReadFile(s.Path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if !bytes.Equal(current, s.snapshot) {
		return errors.New("vault changed in another session; reopen it before saving")
	}
	if err := atomicWrite(s.Path, data); err != nil {
		return err
	}
	s.snapshot = data
	return nil
}

func atomicWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".qv-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func Keys(tokens map[string]string) []string {
	keys := make([]string, 0, len(tokens))
	for key := range tokens {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
