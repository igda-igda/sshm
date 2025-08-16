# SSHM - SSH Connection Manager

[![Go](https://github.com/idabic/sshm/workflows/Go/badge.svg)](https://github.com/idabic/sshm/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/idabic/sshm)](https://goreportcard.com/report/github.com/idabic/sshm)

SSHM is a CLI SSH connection manager that helps DevOps engineers, system administrators, and developers connect to multiple remote servers simultaneously through organized tmux sessions.

## Features

- **Server Configuration Management**: Store and organize SSH connection details with profiles
- **Tmux Session Integration**: Automatic tmux session creation and management for each server connection
- **Multiple Authentication Methods**: Support for SSH keys, password authentication, and SSH agent integration
- **Security-First**: Encrypted credential storage and secure configuration management
- **User-Friendly Interface**: Interactive prompts and helpful error messages with emojis
- **Docker Testing Environment**: Complete testing setup with SSH servers for development

## Installation

### From Source

```bash
git clone https://github.com/idabic/sshm.git
cd sshm
go build -o sshm main.go
sudo mv sshm /usr/local/bin/
```

### Prerequisites

- Go 1.21 or later
- tmux (required for session management)
- SSH client

## Quick Start

### Add a Server

```bash
sshm add production-api
```

This will prompt you for:
- Hostname/IP address
- SSH port (default: 22)
- Username
- Authentication method (key or password)
- SSH key path (if using key authentication)
- Passphrase protection status

### List Servers

```bash
sshm list
```

### Connect to a Server

```bash
sshm connect production-api
```

This will:
- Create a tmux session named after the server
- Execute the SSH connection within the session
- Attach to the session for interactive use

### Remove a Server

```bash
sshm remove production-api
```

## Usage Examples

### Basic Workflow

```bash
# Add a production server with SSH key authentication
sshm add prod-web
# Hostname: web.prod.company.com
# Port (default: 22): 22
# Username: deploy
# Authentication type (key/password): key
# SSH key path: ~/.ssh/prod_rsa
# Is the key passphrase protected? (y/n): n

# List all configured servers
sshm list

# Connect to the server
sshm connect prod-web

# Remove the server when no longer needed
sshm remove prod-web
```

### Multiple Server Management

```bash
# Add multiple servers
sshm add prod-api
sshm add staging-db
sshm add jump-host

# List all servers
sshm list

# Connect to different servers in separate tmux sessions
sshm connect prod-api
sshm connect staging-db
```

## Configuration

Server configurations are stored in `~/.sshm/config.yaml` with file permissions set to 600 for security.

Example configuration:
```yaml
servers:
  - name: "production-api"
    hostname: "api.prod.company.com"
    port: 22
    username: "deploy"
    auth_type: "key"
    key_path: "~/.ssh/prod_rsa"
    passphrase_protected: false
  - name: "staging-db"
    hostname: "db.staging.company.com"
    port: 2222
    username: "admin"
    auth_type: "password"
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/config
go test ./internal/tmux
go test ./cmd
```

### Integration Testing with Docker

The project includes a Docker-based SSH testing environment:

```bash
# Start test SSH servers
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -v integration_test.go

# Clean up
docker-compose -f docker-compose.test.yml down
```

### Project Structure

```
sshm/
├── cmd/                    # CLI commands
│   ├── add.go             # Add server command
│   ├── list.go            # List servers command
│   ├── remove.go          # Remove server command
│   ├── connect.go         # Connect to server command
│   └── root.go            # Root command setup
├── internal/
│   ├── config/            # Configuration management
│   ├── ssh/               # SSH client wrapper
│   └── tmux/              # Tmux session management
├── test/                  # Docker test environment
├── integration_test.go    # Integration tests
└── main.go               # Application entry point
```

## Architecture

### Tmux Session Management

- **Single Server Connections**: Creates a tmux session named after the server
- **Session Isolation**: Each server connection gets its own dedicated tmux session
- **Automatic Cleanup**: Sessions are managed automatically to prevent conflicts

### Security Considerations

- **File Permissions**: Configuration files are created with 600 permissions
- **No Plaintext Passwords**: Passwords are never stored, only authentication methods
- **SSH Key References**: Only file paths to keys are stored, never the keys themselves
- **Host Key Verification**: SSH connections include host key verification options

### Authentication Support

- **SSH Key Files**: Support for private keys with optional passphrase protection
- **SSH Agent Integration**: Seamless integration with existing SSH agent workflows
- **Password Authentication**: Interactive password prompts with secure input
- **Jump Host Support**: Ready for bastion server configurations (future enhancement)

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Roadmap

### Phase 1: Core SSH Management ✅
- [x] Server configuration management
- [x] Basic CLI commands
- [x] Tmux session creation
- [x] SSH key integration
- [x] Password authentication

### Phase 2: Profiles and Group Operations (Future)
- [ ] Server profiles (dev, staging, prod)
- [ ] Group connection support
- [ ] Import/export configurations
- [ ] Enhanced session management

### Phase 3: Interactive TUI Interface (Future)
- [ ] k9s-inspired TUI interface
- [ ] Server browser
- [ ] Session manager
- [ ] Quick actions

### Phase 4: Advanced Features (Future)
- [ ] Encrypted storage
- [ ] Jump host support
- [ ] Connection history
- [ ] Advanced SSH options

### Phase 5: Collaboration (Future)
- [ ] Team configuration templates
- [ ] Cloud provider integration
- [ ] Ansible inventory import

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [k9s](https://github.com/derailed/k9s) for terminal UI patterns
- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Uses [tview](https://github.com/rivo/tview) foundation for future TUI development