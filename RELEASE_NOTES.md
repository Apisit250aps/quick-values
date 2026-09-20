# Quick Vault v2.0.0

Quick Vault 2.0 introduces a pastel terminal interface, optional value encryption, and password authorization for individual actions. Browse key names freely, then authorize access when copying, revealing, or changing a value.

## Highlights

- **Interactive vault:** Open `qv`, `qv list`, or `qv get` to browse, search, and select keys with Bubble Tea v2.
- **Pastel styling:** Shared colors across the vault, forms, help, confirmations, and command output.
- **Optional encryption:** Run `qv init` and set a password to encrypt values with AES-256-GCM and Argon2id. Leave the password empty for plaintext storage. The storage mode is displayed in the UI and command output.
- **Visible keys, protected values:** Encrypted vaults keep key names readable. Copying, revealing, saving, and deleting require a password for each action; browsing does not retain an unlocked session.
- **Private clipboard output:** `qv get <key>` copies without printing the value. Add `--show` to explicitly display it after a successful copy.
- **Password management:** `qv change-password` verifies the current password and re-encrypts existing values with a new password. It can also enable encryption on a plaintext vault.
- **Reset and reinitialize:** `qv reset` requires typing `RESET` before setting up a new, empty vault. Cancelling setup preserves existing data.

## Reliability improvements

- Atomic file replacement and write locking help prevent partial writes and conflicting saves.
- Invalid vault data and incorrect passwords produce errors without overwriting the file.
- Encrypted values are authenticated against their key names; password verification also authenticates the key list.
- Clipboard commands receive values through stdin instead of shell interpolation.
- Clipboard failures are reported instead of displaying a success message.
- Linux clipboard support includes Wayland through `wl-copy`, with `xclip` and `xsel` fallbacks.
- Cancelling a password prompt while saving preserves the form draft.

## Breaking changes

- **Initialization is required.** Run `qv init` before using an uninitialized vault, including an existing v1 plaintext vault.
- **`get` no longer prints values by default.** Use `qv get <key> --show` when visible output is intended.
- **`list` is interactive.** It opens the selector instead of printing a static key list, and requires a terminal.
- **Encrypted operations require interactive password entry.** Passwords are not accepted through command-line flags or environment variables.
- **The storage format has changed.** Older binaries cannot read the new versioned vault format.

## Upgrading

Keep a backup of your existing vault before upgrading. Its location remains `~/.qv` on macOS/Linux or `%USERPROFILE%\.qv` on Windows.

1. Install the v2.0.0 binary for your operating system and processor, or build from the v2.0.0 source with Go 1.27.1 or newer.
2. For an existing v1 plaintext JSON vault, run `qv init`. Existing keys and values are preserved, and you can choose whether to encrypt the values.
3. For a vault created by an earlier build that encrypted the entire key/value map, open `qv`. A one-time password prompt migrates it to value-only encryption while retaining the same password and all values.
4. For an already initialized value-encrypted vault, run `qv` directly.

**Do not use `qv reset` to upgrade:** reset intentionally deletes all saved values. After migration, key names are readable on disk; only values remain encrypted.

## Quick start

Download and extract the archive for your platform from this release:

| Platform | Archive |
| --- | --- |
| macOS Apple Silicon | `qv_2.0.0_darwin_arm64.tar.gz` |
| macOS Intel | `qv_2.0.0_darwin_amd64.tar.gz` |
| Linux amd64 | `qv_2.0.0_linux_amd64.tar.gz` |
| Windows amd64 | `qv_2.0.0_windows_amd64.zip` |

`checksums.txt` contains the SHA-256 checksums for these archives. On macOS, install the extracted binary with:

```bash
sudo install -m 755 qv /usr/local/bin/qv
```

```bash
qv init                       # First-time setup; optional encryption
qv                            # Browse keys without a password
qv set api_token "example"     # Authorize before saving in encrypted mode
qv get api_token               # Authorize, then copy without displaying
qv get api_token --show        # Authorize, copy, and display
qv change-password            # Change the encryption password
```

Use the interactive add/edit form when you want to avoid putting values in shell history. The interactive interface requires a terminal of at least 50 columns by 22 rows.

## Validation

- Automated tests, race checks, and `go vet` passed.
- Terminal flows verified for browsing, per-action authorization, clipboard output, password changes, and reset.
- Whole-vault migration verified by comparing all decrypted key/value pairs before and after migration.
- Built for macOS Apple Silicon and cross-built for macOS Intel, Windows amd64, and Linux amd64. Windows and Linux runtime behavior has not been verified on those operating systems.
