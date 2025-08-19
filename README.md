# SSHM - SSH Connection Manager

[![Go](https://github.com/idabic/sshm/workflows/Go/badge.svg)](https://github.com/idabic/sshm/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/idabic/sshm)](https://goreportcard.com/report/github.com/idabic/sshm)
[![Release](https://img.shields.io/github/v/release/igda-igda/sshm)](https://github.com/igda-igda/sshm/releases)

SSHM is a CLI SSH connection manager that helps DevOps engineers, system administrators, and developers connect to multiple remote servers simultaneously through organized tmux sessions, featuring professional color support and modern terminal interface design.

## ✨ Key Features

### 🖥️ Server Management
- **Server Configuration Management**: Store and organize SSH connection details with profiles
- **Profile-Based Organization**: Group servers by environment (dev, staging, prod) or custom categories
- **Batch Connections**: Connect to multiple servers simultaneously in organized tmux layouts
- **Import/Export Support**: Share configurations via YAML/JSON files for team collaboration

### 🔧 Session Management  
- **Tmux Integration**: Automatic tmux session creation and management for each connection
- **Session Naming Logic**: Server name for single connections, group name for batch connections
- **Window Management**: Individual tmux windows for each server in group sessions
- **Session Persistence**: Resume existing sessions or create new ones as needed

### 🔐 Authentication & Security
- **Multiple Authentication Methods**: SSH keys, password authentication, and SSH agent integration
- **Security-First Design**: Secure credential handling with encrypted storage capabilities
- **SSH Key Management**: Support for passphrase-protected keys and key file references
- **Host Verification**: SSH connection validation and security checks

### 🎨 Modern CLI Experience
- **Professional Color Support**: k9s and kubectl-inspired interface with intelligent color schemes
- **Terminal Compatibility**: Automatic adaptation to different terminal environments
- **NO_COLOR Compliance**: Full support for NO_COLOR accessibility specification
- **Visual Hierarchy**: Clear distinction between commands, flags, examples, and status messages
- **Status Message Colors**: Green (success), Red (error), Yellow (warning), Blue (info)

### 🧪 Development & Testing
- **Comprehensive Test Coverage**: Full test suite for all functionality including color support
- **Docker Testing Environment**: Complete testing setup with SSH servers for development
- **Cross-Platform Support**: Works reliably across different terminal emulators and operating systems

## 🚀 Installation

### Direct Download (Recommended)

Download the latest release for your platform:
- **Linux**: `sshm-linux-amd64`
- **macOS**: `sshm-darwin-amd64` 
- **Windows**: `sshm-windows-amd64.exe`

```bash
# Linux/macOS example
curl -L https://github.com/igda-igda/sshm/releases/latest/download/sshm-linux-amd64 -o sshm
chmod +x sshm
sudo mv sshm /usr/local/bin/
```

### Go Install

```bash
go install github.com/igda-igda/sshm@latest
```

### From Source

```bash
git clone https://github.com/igda-igda/sshm.git
cd sshm
go build -o sshm main.go
sudo mv sshm /usr/local/bin/
```

### Prerequisites

- **tmux** (required for session management)
- **SSH client** (usually pre-installed)
- **Go 1.21+** (only needed for building from source)

## ⚡ Quick Start

### 1. Add Your First Server

```bash
sshm add production-api
```

The modern CLI interface will guide you through:
- **Hostname/IP address**
- **SSH port** (default: 22)  
- **Username**
- **Authentication method** (key or password)
- **SSH key path** (if using key authentication)
- **Passphrase protection status**

### 2. List All Servers

```bash
sshm list                    # List all servers
sshm list --profile prod     # List servers in 'prod' profile
```

### 3. Connect to Servers

```bash
# Single server connection
sshm connect production-api

# Batch connection to all servers in a profile
sshm batch --profile staging
```

### 4. Manage Profiles

```bash
# Create a new profile
sshm profile create development

# Assign servers to profiles
sshm profile assign web-server development
sshm profile assign database development

# List all profiles
sshm profile list
```

### 5. Import/Export Configurations

```bash
# Export your configuration
sshm export servers.yaml

# Import from SSH config or previous export  
sshm import ~/.ssh/config
sshm import servers.yaml
```

## 📚 Usage Examples

### Professional CLI Experience

The modern CLI provides beautiful, colored output with clear visual hierarchy:

```bash
# Professional help screens with colors
sshm --help

# Add servers with visual feedback
sshm add prod-web --hostname web.prod.com --username deploy --auth-type key --key-path ~/.ssh/prod
✅ Server 'prod-web' added successfully!
ℹ️  Use 'sshm connect prod-web' to connect to this server

# Colored status messages for all operations
sshm connect prod-web
ℹ️  Connecting to prod-web (deploy@web.prod.com:22)...
ℹ️  Created tmux session: prod-web
✅ Connected to prod-web successfully!
```

### Profile-Based Organization

```bash
# Create development environment
sshm profile create development
sshm add dev-api --hostname api.dev.com --username dev --auth-type key --key-path ~/.ssh/dev
sshm add dev-db --hostname db.dev.com --username dev --auth-type key --key-path ~/.ssh/dev
sshm profile assign dev-api development
sshm profile assign dev-db development

# Connect to all development servers at once
sshm batch --profile development
ℹ️  Creating group session for profile 'development' with 2 server(s)...
ℹ️  Created group session: development
ℹ️  Created 2 windows for servers
✅ Connected to profile 'development' group session successfully!
```

### Team Collaboration

```bash
# Export configuration for team sharing
sshm export team-servers.yaml --profile production
ℹ️  Exporting profile 'production' with 5 servers
✅ Configuration exported to team-servers.yaml (yaml format)

# Import shared configuration
sshm import team-servers.yaml
✅ Import completed:
  • 5 servers imported
  • 1 profiles imported

# Import from existing SSH config
sshm import ~/.ssh/config
✅ Import completed:
  • 12 servers imported
```

### Terminal Compatibility

SSHM automatically adapts to your terminal environment:

```bash
# Full color support in compatible terminals
sshm list

# Plain text mode when NO_COLOR is set
NO_COLOR=1 sshm list

# Automatic plain text for piped output
sshm list | grep production

# Works with automated tools
TERM=dumb sshm batch --profile ci-servers
```

## ⚙️ Configuration

### Configuration Storage

Server configurations are stored in `~/.sshm/config.yaml` with secure file permissions (600) to protect sensitive data.

### Complete Configuration Example
```yaml
servers:
  - name: "production-api"
    hostname: "api.prod.company.com"
    port: 22
    username: "deploy"
    auth_type: "key"
    key_path: "~/.ssh/prod_rsa"
    passphrase_protected: false
    profile: "production"
  - name: "production-web" 
    hostname: "web.prod.company.com"
    port: 22
    username: "deploy"
    auth_type: "key"
    key_path: "~/.ssh/prod_rsa"
    passphrase_protected: false
    profile: "production"
  - name: "staging-db"
    hostname: "db.staging.company.com"
    port: 2222
    username: "admin"
    auth_type: "password"
    profile: "staging"
  - name: "dev-server"
    hostname: "dev.company.com"
    port: 22
    username: "developer"
    auth_type: "key"
    key_path: "~/.ssh/dev_rsa"
    passphrase_protected: true
    profile: "development"

profiles:
  - name: "production"
    description: "Production environment servers"
    servers:
      - "production-api"
      - "production-web"
  - name: "staging"
    description: "Staging environment for testing"
    servers:
      - "staging-db"
  - name: "development"  
    description: "Development environment"
    servers:
      - "dev-server"
```

### CLI Command Reference

```bash
# Server Management
sshm add <name>                          # Add new server (interactive)
sshm add <name> [flags]                  # Add new server (non-interactive)
sshm list [--profile <name>]             # List servers, optionally filtered by profile
sshm connect <name>                      # Connect to single server
sshm remove <name>                       # Remove server configuration
sshm sessions list                       # List active tmux sessions

# Profile Management  
sshm profile create <name>               # Create new profile
sshm profile list                        # List all profiles with server counts
sshm profile delete <name>               # Delete profile (with confirmation)
sshm profile assign <server> <profile>   # Assign server to profile
sshm profile unassign <server> <profile> # Remove server from profile

# Batch Operations
sshm batch --profile <name>              # Connect to all servers in profile
sshm import <file>                       # Import from YAML/JSON/SSH config
sshm export <file> [--profile <name>]    # Export configuration
sshm export <file> --format json         # Export in JSON format

# Color and Accessibility
NO_COLOR=1 sshm <command>                # Disable colors (accessibility)
TERM=dumb sshm <command>                 # Force plain text for automation
```

## 🛠️ Development

### Running Tests

```bash
# Run all tests with color output
go test ./...

# Run tests with coverage report
go test -cover ./...

# Run specific package tests
go test ./internal/config    # Configuration management tests
go test ./internal/color     # Color support and terminal compatibility tests
go test ./internal/tmux      # Tmux session management tests 
go test ./internal/ssh       # SSH connection handling tests
go test ./cmd               # CLI command tests
```

### Testing Color Support

```bash
# Test color functionality specifically
go test ./internal/color -v

# Test terminal compatibility
NO_COLOR=1 go test ./internal/color
TERM=dumb go test ./internal/color

# Test CLI color formatting
go test ./cmd -v -run="Color"
```

### Integration Testing with Docker

The project includes a comprehensive Docker-based SSH testing environment:

```bash
# Start test SSH servers
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -v integration_test.go

# Test with different terminal environments
NO_COLOR=1 go test -v integration_test.go

# Clean up
docker-compose -f docker-compose.test.yml down
```

### Project Structure

```
sshm/
├── cmd/                       # CLI commands and interfaces
│   ├── add.go                # Server addition with interactive prompts
│   ├── batch.go              # Profile-based batch connections
│   ├── connect.go            # Individual server connections
│   ├── export.go             # Configuration export functionality
│   ├── help_formatter.go     # Color help formatting utilities
│   ├── import.go             # Configuration import from various sources
│   ├── list.go               # Server and profile listing
│   ├── profile.go            # Profile management commands
│   ├── remove.go             # Server removal with confirmations
│   ├── root.go               # Root command with color support
│   ├── sessions.go           # Tmux session management
│   ├── *_test.go             # Comprehensive CLI testing suite
│   └── *_color_test.go       # Color functionality testing
├── internal/
│   ├── color/                # Modern CLI color support
│   │   ├── color.go          # Color utilities with terminal detection
│   │   └── color_test.go     # Terminal compatibility testing
│   ├── config/               # Configuration management
│   │   ├── config.go         # YAML-based configuration handling
│   │   └── *_test.go         # Configuration validation tests
│   ├── ssh/                  # SSH client wrapper
│   │   ├── ssh.go            # SSH connection management
│   │   └── *_test.go         # SSH functionality tests
│   └── tmux/                 # Tmux session management
│       ├── tmux.go           # Session creation and management
│       └── *_test.go         # Tmux integration tests
├── test/                     # Docker test environment
│   ├── docker-compose.test.yml
│   ├── ssh-server/           # Test SSH server configurations
│   └── integration/          # Integration test scenarios
├── .agent-os/                # Agent OS workflow documentation
│   ├── product/              # Product specifications and roadmap
│   └── specs/                # Feature specifications and tasks
├── integration_test.go       # End-to-end integration tests
├── go.mod                    # Go module dependencies
├── go.sum                    # Dependency checksums
└── main.go                   # Application entry point
```

### Building and Testing

```bash
# Build with optimizations
go build -ldflags="-s -w" -o sshm main.go

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o sshm-linux-amd64 main.go
GOOS=darwin GOARCH=amd64 go build -o sshm-darwin-amd64 main.go  
GOOS=windows GOARCH=amd64 go build -o sshm-windows-amd64.exe main.go

# Verify color support works correctly
./sshm --help                    # Should show colors
NO_COLOR=1 ./sshm --help         # Should show plain text
echo "test" | ./sshm --help      # Should auto-detect non-TTY
```

## 🏗️ Architecture

### Modern CLI Design

- **Professional Interface**: k9s and kubectl-inspired design with intelligent color schemes
- **Terminal Adaptation**: Automatic detection of terminal capabilities and appropriate output formatting
- **Accessibility Compliance**: Full NO_COLOR specification support for screen readers and accessibility tools
- **Cross-Platform Compatibility**: Consistent behavior across Linux, macOS, Windows, and different terminal emulators

### Tmux Session Management

- **Single Server Connections**: Creates dedicated tmux sessions named after each server
- **Group Connections**: Profile-based sessions with multiple windows (one per server)
- **Session Naming Logic**: Server name for individual connections, profile name for group sessions
- **Session Persistence**: Automatic detection and reattachment to existing sessions
- **Window Management**: Intuitive navigation between servers in group sessions

### Profile-Based Organization

- **Environment Grouping**: Organize servers by environment (dev, staging, prod) or custom categories
- **Batch Operations**: Connect to all servers in a profile simultaneously
- **Team Collaboration**: Import/export configurations for sharing across teams
- **Flexible Assignment**: Servers can belong to multiple profiles for different use cases

### Security Architecture

- **File Permissions**: Configuration files created with 600 permissions for user-only access
- **No Credential Storage**: Passwords never stored, only authentication method preferences
- **SSH Key References**: Only file paths stored, never private key content
- **Host Verification**: SSH connections include proper host key verification
- **Secure Defaults**: Security-first configuration with safe defaults

### Authentication Systems

- **SSH Key Files**: Full support for private keys with optional passphrase protection
- **SSH Agent Integration**: Seamless integration with existing SSH agent workflows
- **Password Authentication**: Secure interactive password prompts with hidden input
- **Multi-Method Support**: Mix different authentication types across servers

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 🗺️ Roadmap

### Phase 1: Core SSH Management ✅ COMPLETED
- [x] Server configuration management with YAML storage
- [x] Basic CLI commands (add, list, connect, remove)
- [x] Tmux session creation and management
- [x] SSH key integration with passphrase support
- [x] Password authentication with secure prompts
- [x] Connection validation and error handling

### Phase 2: Profiles and Group Operations ✅ COMPLETED
- [x] Server profiles for environment organization (dev, staging, prod)
- [x] Profile-based batch connection support with tmux windows
- [x] Import/export configurations (YAML, JSON, SSH config)
- [x] Enhanced session management with group sessions
- [x] Profile assignment and management commands

### Phase 2.5: CLI Visual Enhancements ✅ COMPLETED
- [x] Professional color support with k9s-inspired design
- [x] Terminal compatibility with NO_COLOR and TTY detection
- [x] Status message colors throughout application
- [x] Enhanced help screens with visual hierarchy
- [x] Cross-platform terminal emulator support

### Phase 3: Interactive TUI Interface (Planning)
- [ ] k9s-inspired TUI interface with tview framework
- [ ] Interactive server browser with search and filtering  
- [ ] Visual session manager for tmux sessions
- [ ] Quick actions (connect, edit, delete) from TUI
- [ ] Real-time connection status indicators

### Phase 4: Advanced Features and Security (Future)
- [ ] Encrypted credential storage using system keyring
- [ ] Jump host/bastion server support with tunneling
- [ ] Connection history tracking and recent connections
- [ ] Advanced SSH options (port forwarding, custom options)
- [ ] Host key verification and MITM protection

### Phase 5: Collaboration and Integrations (Future)
- [ ] Team configuration templates for standardized setups
- [ ] Cloud provider integration (AWS, GCP, Azure instance discovery)
- [ ] Ansible inventory import for existing infrastructure
- [ ] Command broadcasting to multiple servers
- [ ] Enhanced monitoring and connection health checks

## 📈 Current Status

**Latest Release**: v1.2.0 - CLI Visual Enhancements  
**Development Focus**: Preparing for Phase 3 (Interactive TUI Interface)  
**Stability**: Production-ready for daily use with comprehensive testing

**Completed Features**:
- ✅ Full server and profile management
- ✅ Professional CLI with color support  
- ✅ Batch operations and team collaboration
- ✅ Import/export functionality
- ✅ Secure configuration handling
- ✅ Cross-platform terminal compatibility

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

### Design Inspiration
- **[k9s](https://github.com/derailed/k9s)** - Terminal UI patterns and professional CLI design principles
- **[kubectl](https://kubernetes.io/docs/reference/kubectl/)** - Color scheme and command interface design

### Core Technologies
- **[Cobra](https://github.com/spf13/cobra)** - Powerful CLI framework for Go applications
- **[Viper](https://github.com/spf13/viper)** - Configuration management and YAML parsing
- **[fatih/color](https://github.com/fatih/color)** - Cross-platform terminal color support

### Terminal Technologies  
- **[golang.org/x/term](https://pkg.go.dev/golang.org/x/term)** - Terminal capability detection and TTY handling
- **[tview](https://github.com/rivo/tview)** - Future TUI development foundation

### Community Standards
- **[NO_COLOR](https://no-color.org/)** - Accessibility specification for command-line tools

Special thanks to the Go community for creating robust, well-documented libraries that make building professional CLI tools both enjoyable and reliable.