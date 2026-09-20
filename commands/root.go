// Package commands contains CLI commands and the interactive vault UI.
package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"qv/libs"
	"qv/utils"
)

type runner struct {
	store       *libs.Store
	out         io.Writer
	prompt      func(title, hint string, secret bool) (string, error)
	copy        func(string) error
	interactive func(libs.Store, map[string]string) error
}

func Run(args []string, out io.Writer) error {
	store, err := libs.DefaultStore()
	if err != nil {
		return err
	}
	defer store.Close()
	r := runner{store: &store, out: out, copy: utils.Copy}
	r.prompt = func(title, hint string, secret bool) (string, error) { return ask(out, title, hint, secret) }
	r.interactive = func(store libs.Store, values map[string]string) error {
		if !term.IsTerminal(os.Stdin.Fd()) {
			return errors.New("interactive vault requires a terminal; use qv get <key> to copy a value")
		}
		_, err := tea.NewProgram(newModel(store, values), tea.WithOutput(out)).Run()
		return err
	}
	return r.run(args)
}

func validate(args []string) error {
	if len(args) == 0 {
		return nil
	}
	switch args[0] {
	case "help", "--help", "-h", "init", "reset", "change-password", "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: qv %s", args[0])
		}
	case "set":
		if len(args) != 3 || strings.TrimSpace(args[1]) == "" {
			return errors.New("usage: qv set <key> <value>")
		}
	case "get":
		if len(args) != 1 && len(args) != 2 && !(len(args) == 3 && args[2] == "--show") {
			return errors.New("usage: qv get [<key> [--show]]")
		}
	case "del":
		if len(args) != 2 {
			return errors.New("usage: qv del <key>")
		}
	default:
		return fmt.Errorf("unknown command: %s (use qv help)", args[0])
	}
	return nil
}

func (r runner) run(args []string) error {
	if err := validate(args); err != nil {
		return err
	}
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "help", "--help", "-h":
		showHelp(r.out)
		return nil
	case "init":
		return r.initialize(false)
	case "reset":
		return r.initialize(true)
	}
	status, err := r.store.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return libs.ErrNotInitialized
	}
	if cmd == "change-password" {
		return r.changePassword(status.Encrypted)
	}
	if status.NeedsMigration {
		password, err := r.prompt("Migrate existing vault", "Unlock once to make key names visible; values stay encrypted", true)
		if err != nil {
			return err
		}
		if err := r.store.Migrate(password); err != nil {
			return err
		}
		success(r.out, "Vault migrated. Key names are visible; values remain encrypted.")
	}
	names, err := r.store.Browse()
	if err != nil {
		return err
	}
	if cmd == "" || cmd == "list" || cmd == "get" && len(args) == 1 {
		return r.interactive(*r.store, names)
	}
	if cmd == "get" || cmd == "del" {
		if _, ok := names[args[1]]; !ok {
			return fmt.Errorf("key not found: %s", args[1])
		}
	}
	password := ""
	if status.Encrypted {
		password, err = r.prompt("Authorize action", "Enter your password to copy, display, or change a value", true)
		if err != nil {
			return err
		}
	}
	values, err := r.store.Unlock(password)
	password = ""
	if err != nil {
		return err
	}
	defer r.store.Close()
	storageStatus(r.out, status.Encrypted)
	switch cmd {
	case "set":
		values[args[1]] = args[2]
		if err := r.store.Save(values); err != nil {
			return err
		}
		success(r.out, "Saved "+args[1])
	case "get":
		value, ok := values[args[1]]
		if !ok {
			return fmt.Errorf("key not found: %s", args[1])
		}
		if err := r.copy(value); err != nil {
			return err
		}
		success(r.out, "Copied "+args[1]+" to clipboard")
		if len(args) == 3 {
			fmt.Fprintln(r.out, sky.Render(safe(value)))
		}
	case "del":
		if _, ok := values[args[1]]; !ok {
			return fmt.Errorf("key not found: %s", args[1])
		}
		delete(values, args[1])
		if err := r.store.Save(values); err != nil {
			return err
		}
		success(r.out, "Deleted "+args[1])
	}
	return nil
}

func (r runner) newPassword(optional bool) (string, error) {
	hint := "Choose a new password (required)"
	if optional {
		hint = "Password encrypts your vault. Leave empty for plain text."
	}
	password, err := r.prompt("Set encryption password", hint, true)
	if err != nil {
		return "", err
	}
	if password == "" {
		if optional {
			return "", nil
		}
		return "", errors.New("new password must not be empty")
	}
	confirmation, err := r.prompt("Confirm password", "Enter the same password again", true)
	if err != nil {
		return "", err
	}
	if password != confirmation {
		return "", errors.New("passwords do not match; vault has not been changed")
	}
	return password, nil
}

func (r runner) initialize(reset bool) error {
	if reset {
		answer, err := r.prompt("Reset Quick Vault", "All saved values and settings will be deleted. Type RESET to continue.", false)
		if err != nil {
			return err
		}
		if answer != "RESET" {
			return errCancelled
		}
	} else {
		status, err := r.store.Status()
		if err != nil {
			return err
		}
		if status.Initialized {
			return errors.New("vault is already initialized; use qv change-password or qv reset")
		}
		if status.Legacy {
			fmt.Fprintln(r.out, peach.Render("Existing values will be preserved and migrated during setup."))
		}
	}
	password, err := r.newPassword(true)
	if err != nil {
		return err
	}
	if err := r.store.Initialize(password, reset); err != nil {
		return err
	}
	if reset {
		success(r.out, "Previous data removed. New vault initialized.")
	} else {
		success(r.out, "Vault initialized. Run qv to get started.")
	}
	storageStatus(r.out, r.store.Encrypted())
	return nil
}

func (r runner) changePassword(encrypted bool) error {
	oldPassword := ""
	var err error
	if encrypted {
		oldPassword, err = r.prompt("Current password", "Unlock existing data before changing its password", true)
		if err != nil {
			return err
		}
	} else {
		fmt.Fprintln(r.out, peach.Render("This vault has no password. Set one to encrypt existing values."))
	}
	// Validate the current password before asking for a replacement.
	if _, err := r.store.Unlock(oldPassword); err != nil {
		return err
	}
	password, err := r.newPassword(false)
	if err != nil {
		return err
	}
	if err := r.store.ChangePassword(oldPassword, password); err != nil {
		return err
	}
	success(r.out, "Password changed. All values re-encrypted.")
	storageStatus(r.out, true)
	return nil
}
