# QV - Quick Vault 🔐

A lightweight command-line tool for securely storing and quickly retrieving tokens, API keys, and other sensitive values with automatic clipboard integration.

## Features

- **Simple Storage**: Store key-value pairs securely in your home directory
- **Quick Access**: Retrieve values instantly and copy to clipboard automatically
- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Secure**: Files are stored with restricted permissions (0600)
- **No Dependencies**: Single binary with no external requirements

## Installation

### From Source

```bash
git clone https://github.com/Apisit250aps/qv.git
cd qv
go build -o qv main.go
```

### Manual Installation

1. Download the binary for your platform from the [releases page](https://github.com/yourusername/qv/releases)
2. Make it executable: `chmod +x qv`
3. Move to your PATH: `mv qv /usr/local/bin/` (or add to your PATH)

## Usage

### Commands

```bash
qv set <key> <value>    # Store a new token
qv get <key>            # Retrieve and copy token to clipboard
qv del <key>            # Delete a stored token
qv list                 # List all available keys
```

### Examples

```bash
# Store an API key
qv set github_token ghp_xxxxxxxxxxxxxxxxxxxx

# Store a database password
qv set prod_db_pass "my-secure-password-123"

# Retrieve a token (automatically copies to clipboard)
qv get github_token
# Output: github_token: ghp_xxxxxxxxxxxxxxxxxxxx
#         (Copied to clipboard)

# List all stored keys
qv list
# Output: Available keys:
#         - github_token
#         - prod_db_pass

# Delete a token
qv del github_token
# Output: Deleted github_token successfully.
```

## Storage Location

Tokens are stored in a JSON file at:
- **Linux/macOS**: `~/.qv`
- **Windows**: `%USERPROFILE%\.qv`

The file is created with restricted permissions (0600) for security.

## Security Considerations

- Tokens are stored in plain text JSON format
- Files are created with user-only read/write permissions (0600)
- This tool is designed for development convenience, not enterprise-grade secret management
- For production environments, consider using dedicated secret management solutions

## Clipboard Support

QV automatically copies retrieved tokens to your clipboard:

- **Windows**: Uses `clip` command
- **macOS**: Uses `pbcopy` command  
- **Linux**: Uses `xclip` or `xsel` (fallback)

### Linux Clipboard Requirements

On Linux, you may need to install clipboard utilities:

```bash
# Ubuntu/Debian
sudo apt-get install xclip

# Or alternatively
sudo apt-get install xsel

# Fedora/RHEL
sudo dnf install xclip

# Arch Linux
sudo pacman -S xclip
```

## Use Cases

- **API Keys**: Store GitHub, AWS, or other service tokens
- **Database Credentials**: Quick access to development database passwords
- **SSH Keys**: Store SSH key passphrases or connection strings
- **Development Secrets**: Any sensitive values needed during development

## Development

### Building from Source

```bash
git clone https://github.com/yourusername/qv.git
cd qv
go mod init qv
go build -o qv main.go
```

### Testing

```bash
# Test basic functionality
./qv set test_key test_value
./qv get test_key
./qv list
./qv del test_key
```

### Cross-Platform Builds

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o qv.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o qv-darwin main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o qv-linux main.go
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Troubleshooting

### Common Issues

**"Command not found"**
- Ensure the binary is in your PATH
- Check if the binary has execute permissions

**"Cannot determine home directory"**
- Ensure your HOME environment variable is set correctly

**"Clipboard not supported on this OS"**
- You're running on an unsupported operating system
- Tokens will still be displayed but not copied to clipboard

**Linux clipboard not working**
- Install `xclip` or `xsel` packages
- Ensure you're running in a graphical environment with clipboard support

## Changelog

### v1.0.0
- Initial release
- Basic set/get/delete/list functionality
- Cross-platform clipboard support
- Secure file storage with restricted permissions

---

**Warning**: This tool stores secrets in plain text. Use responsibly and ensure your system is secure.