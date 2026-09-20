// Package commands contains CLI commands and the interactive vault UI.
package commands

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"io"
	"qv/libs"
	"qv/utils"
	"strings"
)

const usage = `Quick Vault · keep your values close

  qv                     Open interactive vault
  qv set <key> <value>    Store or replace a value
  qv get <key>            Print and copy a value
  qv del <key>            Delete a value
  qv list                List keys
  qv help                Show help
`

func Run(args []string, out io.Writer) error {
	if len(args) > 0 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		_, err := fmt.Fprint(out, usage)
		return err
	}
	store, err := libs.DefaultStore()
	if err != nil {
		return err
	}
	tokens, err := store.Load()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		_, err := tea.NewProgram(newModel(store, tokens), tea.WithOutput(out)).Run()
		return err
	}
	switch args[0] {
	case "set":
		if len(args) != 3 || strings.TrimSpace(args[1]) == "" {
			return fmt.Errorf("usage: qv set <key> <value>")
		}
		tokens[args[1]] = args[2]
		if err := store.Save(tokens); err != nil {
			return err
		}
		fmt.Fprintln(out, "Set", args[1], "successfully.")
	case "get":
		if len(args) != 2 {
			return fmt.Errorf("usage: qv get <key>")
		}
		value, ok := tokens[args[1]]
		if !ok {
			return fmt.Errorf("key not found: %s", args[1])
		}
		fmt.Fprintln(out, args[1]+":", value)
		if err := utils.Copy(value); err != nil {
			return err
		}
		fmt.Fprintln(out, "(Copied to clipboard)")
	case "del":
		if len(args) != 2 {
			return fmt.Errorf("usage: qv del <key>")
		}
		if _, ok := tokens[args[1]]; !ok {
			return fmt.Errorf("key not found: %s", args[1])
		}
		delete(tokens, args[1])
		if err := store.Save(tokens); err != nil {
			return err
		}
		fmt.Fprintln(out, "Deleted", args[1], "successfully.")
	case "list":
		if len(args) != 1 {
			return fmt.Errorf("usage: qv list")
		}
		if len(tokens) == 0 {
			fmt.Fprintln(out, "No tokens saved.")
			return nil
		}
		fmt.Fprintln(out, "Available keys:")
		for _, key := range libs.Keys(tokens) {
			fmt.Fprintln(out, "-", key)
		}
	default:
		return fmt.Errorf("unknown command: %s (use qv help)", args[0])
	}
	return nil
}
