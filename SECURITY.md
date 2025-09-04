# Security Policy

## Supported Versions

SSHM follows semantic versioning. We actively support the latest major version and provide security updates as needed.

| Version | Supported          | Status |
| ------- | ------------------ | ------ |
| 1.4.x   | :white_check_mark: | Active development |
| < 1.4   | :x:                | No longer supported |

## Reporting a Vulnerability

SSHM handles sensitive SSH credentials and server connections. If you discover a security vulnerability, please report it responsibly.

### How to Report

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via:
- GitHub's private vulnerability reporting feature at: https://github.com/igda-igda/sshm/security/advisories/new
- Email: Create a GitHub issue marked as "Security" if private reporting is unavailable

### What to Include in Your Report

Please provide as much information as possible:

- **Vulnerability Type**: Buffer overflow, credential exposure, privilege escalation, etc.
- **Affected Components**: Which SSHM features are impacted (TUI, CLI, config storage, SSH connections, etc.)
- **Source Files**: Full paths to affected Go source files
- **Version Information**: SSHM version and Go version used
- **System Information**: OS, architecture, and relevant environment details
- **Reproduction Steps**: Clear, step-by-step instructions to reproduce the issue
- **Impact Assessment**: How an attacker could exploit this vulnerability
- **Proof of Concept**: Code, commands, or screenshots demonstrating the issue (if applicable)

### Response Timeline

- **48 hours**: Acknowledgment of your report
- **72 hours**: Initial assessment and severity classification
- **1 week**: Detailed response with fix timeline or additional questions
- **Regular updates**: Progress reports every 1-2 weeks until resolution

## Security Architecture

SSHM is designed with security as a core principle:

### Credential Security
- **Encrypted Storage**: All sensitive data encrypted using system keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- **No Private Key Storage**: SSH private keys are never stored in SSHM configuration - only file paths are referenced
- **Secure Input**: Password prompts use secure input methods that don't echo to terminal or store in shell history
- **Memory Protection**: Sensitive data cleared from memory after use

### SSH Security
- **Agent Integration**: Seamless integration with SSH agent for key management
- **Connection Validation**: Pre-connection testing to verify server accessibility
- **Multiple Auth Methods**: Support for SSH keys (with optional passphrases), passwords, and SSH agent

### Local Security
- **File Permissions**: Configuration files stored with restricted permissions (600/700)
- **Configuration Encryption**: Server configurations encrypted at rest
- **No Network Communication**: SSHM operates locally only - no data transmitted to external servers
- **Process Isolation**: Each SSH connection runs in isolated tmux sessions

### Session Security
- **Session Cleanup**: Automatic cleanup of orphaned tmux sessions
- **Connection Timeouts**: Configurable timeouts for inactive connections
- **Audit Trail**: Connection history for security monitoring

## Known Security Considerations

### Current Limitations
- **Host Key Verification**: Currently uses `InsecureIgnoreHostKey()` - host key verification not yet implemented
- **Tmux Dependency**: Security depends on tmux's session isolation
- **Local File Access**: Configuration files readable by user account
- **Terminal History**: Commands may appear in shell history (use `HISTIGNORE` to exclude)

### Best Practices for Users
1. **Protect Configuration Directory**: Ensure `~/.sshm/` has proper permissions
2. **Use SSH Agent**: Prefer SSH agent over password authentication
3. **Regular Updates**: Keep SSHM updated to latest version
4. **Verify Connections Manually**: Since host key verification is not implemented, manually verify first connections
5. **Secure Workstation**: Use encrypted storage and screen locks
6. **Network Security**: Use SSHM only on trusted networks until host key verification is implemented

## Security Updates

- Critical security fixes are released immediately as patch versions
- Security advisories published for all confirmed vulnerabilities
- Users notified through GitHub releases and repository security alerts
- Backward compatibility maintained unless security requires breaking changes

## Contact

For security-related questions or concerns:
- Use GitHub's private vulnerability reporting
- Create a confidential issue in the repository
- Engage with maintainers through established project channels

---

**Thank you for helping keep SSHM and the SSH ecosystem secure!**
