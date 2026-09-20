# Quick Vault 🔐

A pastel terminal vault built with Bubble Tea v2. Store values locally, search with the keyboard, and copy without printing secrets.

## Install

Build with Go 1.27.1 or newer:

```bash
git clone https://github.com/Apisit250aps/quick-values.git
cd quick-values
go build -o qv .
```

On macOS, install a binary built for your processor (`darwin-arm64` for Apple Silicon or `darwin-amd64` for Intel):

```bash
chmod +x qv
sudo install -m 755 qv /usr/local/bin/qv
qv init
```

A compiled binary does not require Go on the user's machine.

## First-time setup

Run `qv init` before using the vault. Help is always available.

- Enter and confirm a password to encrypt the vault.
- Leave the password empty to store ordinary, unencrypted values.
- Setup and subsequent commands show the storage mode. The interactive screen keeps this status visible.
- Existing legacy JSON values in `~/.qv` are preserved and migrated during initialization. Running `init` again cannot overwrite an initialized vault.

Passwords are entered through a masked terminal prompt, never command-line flags or environment variables. Opening `qv`, `qv list`, or `qv get` browses key names without a password. Copying, revealing, saving, and deleting each ask for the password separately. The UI does not cache an unlocked session.

Old vaults that encrypted the entire key/value map need a one-time password prompt on first opening. This migrates their key names to readable form and preserves all values encrypted with the same password. Failed authentication leaves the original file untouched.

## Commands

| Command | Behavior |
| --- | --- |
| `qv` | Open the interactive vault |
| `qv set <key> <value>` | Store or replace a value; never print its value |
| `qv get` | Browse and select key names without a password |
| `qv get <key>` | Ask for the password, then copy without displaying the value |
| `qv get <key> --show` | Copy, then explicitly display the value |
| `qv del <key>` | Delete the key |
| `qv list` | Open the same searchable, selectable interface as `qv` |
| `qv init` | Initialize with optional password encryption |
| `qv reset` | Confirm deletion, then initialize a new empty vault |
| `qv change-password` | Verify the old password and re-encrypt all values with a confirmed new password |
| `qv help` | Show colorful command help |

`qv`, `qv list`, setup, and password prompts require an interactive terminal. Plaintext `set`, `get`, and `del` also work without one. Terminal control characters are sanitized in displayed text, including `--show`; clipboard contents retain the exact original value.

```bash
qv init
qv set github_token "example-token"
qv get github_token          # ✓ Copied github_token to clipboard
qv get github_token --show   # Also displays example-token
qv list
qv del github_token
```

A clipboard error produces a failure status rather than claiming the copy succeeded. `set` only stores the value; it does not copy it.

## Interactive controls

| Key | Action |
| --- | --- |
| ↑ / ↓ or j / k | Select a key |
| / | Filter keys; Enter finishes searching |
| Enter / c | Copy; hide any previously revealed value |
| a / e | Add / edit a value |
| Tab / Enter | Move between fields / save |
| r | Explicitly reveal or hide the selected value |
| d | Request deletion; y confirms, n cancels |
| Esc | Cancel an action or clear the filter |
| q / Ctrl+C | Quit; Ctrl+C also works in forms |

Values and password inputs are masked by default. Editing opens an empty replacement-value field; saving asks for authorization. Cancelling authorization keeps the draft. Use a terminal at least 50 columns by 22 rows.

## Reset and password changes

`qv reset` requires typing `RESET`, then asks for the new optional password. The existing vault is replaced only after setup is complete. Cancelling or entering mismatched passwords preserves it. Reset also works if you forgot the old password, but deletes all stored values.

`qv change-password` verifies the old password before asking for the new one. It decrypts the existing values and writes them with a new salt, key, and nonce. A wrong password or failed write preserves the original file. On a plaintext vault, this command sets its first password and encrypts existing values. A new password must not be empty.

## Storage and encryption

The vault is stored at `~/.qv` on macOS/Linux or `%USERPROFILE%\.qv` on Windows. The file starts with `QV1` and contains a versioned JSON envelope.

- **Encrypted mode:** Key names are readable; each value is independently encrypted with AES-256-GCM and bound to its key name. The key list is authenticated when a password is supplied. Argon2id derives a 256-bit key from the password and a random 16-byte salt (3 iterations, 64 MiB memory, 4 lanes). Each encrypted value and authentication record uses a new random nonce on each write. No password is saved.
- **Plaintext mode:** Key names and values are readable in the envelope's `values` object.
- Files are atomically replaced using temporary files with mode `0600`. Windows access is also governed by the directory's ACLs.
- A temporary `~/.qv.lock` prevents simultaneous writes. Sessions with stale data must reopen the vault instead of overwriting newer changes.

Encryption protects the file at rest. Values remain accessible in the running process and on the clipboard after copying. There is no password recovery: keep your password and backups safe. Values passed to `qv set` can appear in shell history; use the interactive add/edit form when that matters. Migrating to encryption does not erase copies in backups or guarantee physical erasure of previously unencrypted disk blocks.

## Clipboard support

- macOS: `pbcopy`
- Windows: `clip.exe`
- Linux: `wl-copy`, then `xclip -selection clipboard`, then `xsel --clipboard --input`

Clipboard utilities receive values through stdin, without shell interpolation. On Linux, install a utility appropriate for your desktop session (for example `wl-clipboard` on Wayland).

## Development

```text
main.go                 Entry point and styled errors
commands/root.go        Command routing and setup workflows
commands/tui.go         Interactive vault
commands/actions.go     Per-action authentication and execution
commands/prompt.go      Masked password and confirmation prompts
commands/theme.go       Shared pastel styles
libs/store.go           Versioned storage, encryption, atomic writes
utils/clipboard.go      Platform clipboard integration
```

```bash
go test ./...
go vet ./...
go build -o qv .

GOOS=darwin GOARCH=arm64 go build -o qv-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o qv-darwin-amd64 .
GOOS=windows GOARCH=amd64 go build -o qv-windows-amd64.exe .
GOOS=linux GOARCH=amd64 go build -o qv-linux-amd64 .
```

If a process is forcibly terminated during a write, a lock file can remain. Only after checking no other `qv` process is writing, remove `~/.qv.lock` and reopen the vault.
