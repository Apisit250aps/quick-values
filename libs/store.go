// Package libs implements vault persistence.
package libs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Store struct{ Path string }

func DefaultStore() (Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Store{}, err
	}
	return Store{Path: filepath.Join(home, ".qv")}, nil
}

func (s Store) Load() (map[string]string, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read vault: %w", err)
	}
	tokens := map[string]string{}
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("invalid vault JSON: %w", err)
	}
	if tokens == nil {
		return nil, fmt.Errorf("vault must be a JSON object")
	}
	return tokens, nil
}

// Save replaces the vault atomically with a file readable only by its owner.
func (s Store) Save(tokens map[string]string) error {
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.Path), ".qv-*")
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
	return os.Rename(f.Name(), s.Path)
}

func Keys(tokens map[string]string) []string {
	keys := make([]string, 0, len(tokens))
	for key := range tokens {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
