package tmux

import (
  "fmt"
  "os/exec"
  "testing"
)

func TestIsAvailable(t *testing.T) {
  tests := []struct {
    name     string
    mockCmd  func(name string, arg ...string) *exec.Cmd
    expected bool
  }{
    {
      name: "tmux available",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "tmux available")
      },
      expected: true,
    },
    {
      name: "tmux not available",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        cmd := exec.Command("false")
        return cmd
      },
      expected: false,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      result := manager.IsAvailable()
      if result != tt.expected {
        t.Errorf("IsAvailable() = %v, want %v", result, tt.expected)
      }
    })
  }
}

func TestCreateSession(t *testing.T) {
  tests := []struct {
    name        string
    sessionName string
    mockCmd     func(name string, arg ...string) *exec.Cmd
    expectError bool
  }{
    {
      name:        "successful session creation",
      sessionName: "test-server",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "session created")
      },
      expectError: false,
    },
    {
      name:        "session creation failure",
      sessionName: "test-server",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("false")
      },
      expectError: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      err := manager.CreateSession(tt.sessionName)
      if (err != nil) != tt.expectError {
        t.Errorf("CreateSession() error = %v, expectError %v", err, tt.expectError)
      }
    })
  }
}

func TestNormalizeSessionName(t *testing.T) {
  tests := []struct {
    name     string
    input    string
    expected string
  }{
    {
      name:     "no special characters",
      input:    "server1",
      expected: "server1",
    },
    {
      name:     "dots converted to underscores",
      input:    "cloudcrafters.cloud",
      expected: "cloudcrafters_cloud",
    },
    {
      name:     "multiple dots",
      input:    "api.staging.company.com",
      expected: "api_staging_company_com",
    },
    {
      name:     "mixed characters",
      input:    "my.server-name",
      expected: "my_server-name",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      result := normalizeSessionName(tt.input)
      if result != tt.expected {
        t.Errorf("normalizeSessionName() = %v, want %v", result, tt.expected)
      }
    })
  }
}

func TestGenerateUniqueSessionName(t *testing.T) {
  tests := []struct {
    name           string
    baseName       string
    existingSessions []string
    expected       string
  }{
    {
      name:             "no conflicts",
      baseName:         "server1",
      existingSessions: []string{},
      expected:         "server1",
    },
    {
      name:             "single conflict",
      baseName:         "server1",
      existingSessions: []string{"server1"},
      expected:         "server1-1",
    },
    {
      name:             "multiple conflicts",
      baseName:         "server1",
      existingSessions: []string{"server1", "server1-1", "server1-2"},
      expected:         "server1-3",
    },
    {
      name:             "non-sequential conflicts",
      baseName:         "server1",
      existingSessions: []string{"server1", "server1-2", "server1-5"},
      expected:         "server1-1",
    },
    {
      name:             "normalize session name with dots",
      baseName:         "cloudcrafters.cloud",
      existingSessions: []string{},
      expected:         "cloudcrafters_cloud",
    },
    {
      name:             "normalize and handle conflicts",
      baseName:         "cloudcrafters.cloud",
      existingSessions: []string{"cloudcrafters_cloud"},
      expected:         "cloudcrafters_cloud-1",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      manager := &Manager{
        existingSessions: tt.existingSessions,
      }
      result := manager.generateUniqueSessionName(tt.baseName)
      if result != tt.expected {
        t.Errorf("generateUniqueSessionName() = %v, want %v", result, tt.expected)
      }
    })
  }
}

func TestListSessions(t *testing.T) {
  tests := []struct {
    name     string
    mockCmd  func(name string, arg ...string) *exec.Cmd
    expected []string
    wantErr  bool
  }{
    {
      name: "list sessions success",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "session1\nsession2\nserver1-1")
      },
      expected: []string{"session1", "session2", "server1-1"},
      wantErr:  false,
    },
    {
      name: "no sessions",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "")
      },
      expected: []string{},
      wantErr:  false,
    },
    {
      name: "tmux list error",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("false")
      },
      expected: nil,
      wantErr:  true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      sessions, err := manager.ListSessions()
      if (err != nil) != tt.wantErr {
        t.Errorf("ListSessions() error = %v, wantErr %v", err, tt.wantErr)
        return
      }
      if !stringSliceEqual(sessions, tt.expected) {
        t.Errorf("ListSessions() = %v, want %v", sessions, tt.expected)
      }
    })
  }
}

func TestSendKeys(t *testing.T) {
  tests := []struct {
    name        string
    sessionName string
    command     string
    mockCmd     func(name string, arg ...string) *exec.Cmd
    expectError bool
  }{
    {
      name:        "send keys success",
      sessionName: "test-session",
      command:     "ssh user@host",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "keys sent")
      },
      expectError: false,
    },
    {
      name:        "send keys failure",
      sessionName: "test-session",
      command:     "ssh user@host",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("false")
      },
      expectError: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      err := manager.SendKeys(tt.sessionName, tt.command)
      if (err != nil) != tt.expectError {
        t.Errorf("SendKeys() error = %v, expectError %v", err, tt.expectError)
      }
    })
  }
}

func TestAttachSession(t *testing.T) {
  tests := []struct {
    name        string
    sessionName string
    mockCmd     func(name string, arg ...string) *exec.Cmd
    expectError bool
  }{
    {
      name:        "attach success",
      sessionName: "test-session",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "attached")
      },
      expectError: false,
    },
    {
      name:        "attach failure",
      sessionName: "test-session",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("false")
      },
      expectError: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      err := manager.AttachSession(tt.sessionName)
      if (err != nil) != tt.expectError {
        t.Errorf("AttachSession() error = %v, expectError %v", err, tt.expectError)
      }
    })
  }
}

func TestConnectToServer(t *testing.T) {
  tests := []struct {
    name            string
    serverName      string
    sshCommand      string
    mockCmd         func(name string, arg ...string) *exec.Cmd
    expectError     bool
    expectedSession string
    expectedExisting bool
  }{
    {
      name:       "connect to new server",
      serverName: "production-api",
      sshCommand: "ssh deploy@api.prod.company.com",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "connected")
      },
      expectError:      false,
      expectedSession:  "production-api",
      expectedExisting: false,
    },
    {
      name:       "reattach to existing session",
      serverName: "production-api",
      sshCommand: "ssh deploy@api.prod.company.com",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "connected")
      },
      expectError:      false,
      expectedSession:  "production-api",
      expectedExisting: true,
    },
    {
      name:       "reattach to existing session with dots",
      serverName: "cloudcrafters.cloud",
      sshCommand: "ssh user@cloudcrafters.cloud",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "connected")
      },
      expectError:      false,
      expectedSession:  "cloudcrafters_cloud",
      expectedExisting: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      if tt.name == "reattach to existing session" {
        manager.existingSessions = []string{"production-api"}
      } else if tt.name == "reattach to existing session with dots" {
        manager.existingSessions = []string{"cloudcrafters_cloud"}
      }

      sessionName, wasExisting, err := manager.ConnectToServer(tt.serverName, tt.sshCommand)
      if (err != nil) != tt.expectError {
        t.Errorf("ConnectToServer() error = %v, expectError %v", err, tt.expectError)
        return
      }
      if sessionName != tt.expectedSession {
        t.Errorf("ConnectToServer() sessionName = %v, want %v", sessionName, tt.expectedSession)
      }
      if wasExisting != tt.expectedExisting {
        t.Errorf("ConnectToServer() wasExisting = %v, want %v", wasExisting, tt.expectedExisting)
      }
    })
  }
}

// Mock Server implementation for testing
type mockServer struct {
  name     string
  hostname string
  port     int
  username string
  authType string
  keyPath  string
  valid    bool
}

func (s *mockServer) GetName() string     { return s.name }
func (s *mockServer) GetHostname() string { return s.hostname }
func (s *mockServer) GetPort() int        { return s.port }
func (s *mockServer) GetUsername() string { return s.username }
func (s *mockServer) GetAuthType() string { return s.authType }
func (s *mockServer) GetKeyPath() string  { return s.keyPath }

func (s *mockServer) Validate() error {
  if !s.valid {
    return fmt.Errorf("invalid server configuration")
  }
  return nil
}

func TestConnectToProfile(t *testing.T) {
  tests := []struct {
    name                string
    profileName         string
    servers             []Server
    mockCmd             func(name string, arg ...string) *exec.Cmd
    existingSessions    []string
    expectError         bool
    expectedSession     string
    expectedExisting    bool
  }{
    {
      name:        "create new profile session with multiple servers",
      profileName: "development",
      servers: []Server{
        &mockServer{name: "web1", hostname: "web1.dev.com", port: 22, username: "dev", authType: "key", keyPath: "~/.ssh/id_rsa", valid: true},
        &mockServer{name: "db1", hostname: "db1.dev.com", port: 22, username: "dev", authType: "key", keyPath: "~/.ssh/id_rsa", valid: true},
        &mockServer{name: "cache1", hostname: "cache1.dev.com", port: 22, username: "dev", authType: "key", keyPath: "~/.ssh/id_rsa", valid: true},
      },
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "success")
      },
      existingSessions:    []string{},
      expectError:         false,
      expectedSession:     "development",
      expectedExisting:    false,
    },
    {
      name:        "reattach to existing profile session",
      profileName: "staging",
      servers: []Server{
        &mockServer{name: "app1", hostname: "app1.staging.com", port: 22, username: "staging", authType: "key", keyPath: "~/.ssh/id_rsa", valid: true},
        &mockServer{name: "app2", hostname: "app2.staging.com", port: 22, username: "staging", authType: "key", keyPath: "~/.ssh/id_rsa", valid: true},
      },
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "success")
      },
      existingSessions:    []string{"staging"},
      expectError:         false,
      expectedSession:     "staging",
      expectedExisting:    true,
    },
    {
      name:        "profile name with dots normalized",
      profileName: "production.api",
      servers: []Server{
        &mockServer{name: "api1", hostname: "api1.prod.com", port: 22, username: "prod", authType: "key", keyPath: "~/.ssh/id_rsa", valid: true},
      },
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "success")
      },
      existingSessions:    []string{},
      expectError:         false,
      expectedSession:     "production_api",
      expectedExisting:    false,
    },
    {
      name:        "session name conflict resolution - reattach to existing",
      profileName: "dev",
      servers: []Server{
        &mockServer{name: "test1", hostname: "test1.dev.com", port: 22, username: "dev", authType: "key", keyPath: "~/.ssh/id_rsa", valid: true},
      },
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "success")
      },
      existingSessions:    []string{"dev", "dev-1"},
      expectError:         false,
      expectedSession:     "dev",
      expectedExisting:    true,
    },
    {
      name:        "session name conflict resolution - create unique name",
      profileName: "newprofile",
      servers: []Server{
        &mockServer{name: "test1", hostname: "test1.dev.com", port: 22, username: "dev", authType: "key", keyPath: "~/.ssh/id_rsa", valid: true},
      },
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "success")
      },
      existingSessions:    []string{"newprofile", "newprofile-1"},
      expectError:         false,
      expectedSession:     "newprofile",
      expectedExisting:    true,
    },
    {
      name:        "server validation error",
      profileName: "test",
      servers: []Server{
        &mockServer{name: "invalid", hostname: "", port: 22, username: "", authType: "key", keyPath: "", valid: false},
      },
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "success")
      },
      existingSessions:    []string{},
      expectError:         true,
      expectedSession:     "",
      expectedExisting:    false,
    },
    {
      name:        "tmux session creation failure",
      profileName: "failed",
      servers: []Server{
        &mockServer{name: "server1", hostname: "server1.com", port: 22, username: "user", authType: "key", keyPath: "~/.ssh/id_rsa", valid: true},
      },
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        if len(arg) > 0 && arg[0] == "new-session" {
          return exec.Command("false") // Fail session creation
        }
        return exec.Command("echo", "success")
      },
      existingSessions:    []string{},
      expectError:         true,
      expectedSession:     "",
      expectedExisting:    false,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{
        existingSessions: tt.existingSessions,
      }

      sessionName, wasExisting, err := manager.ConnectToProfile(tt.profileName, tt.servers)
      if (err != nil) != tt.expectError {
        t.Errorf("ConnectToProfile() error = %v, expectError %v", err, tt.expectError)
        return
      }
      if sessionName != tt.expectedSession {
        t.Errorf("ConnectToProfile() sessionName = %v, want %v", sessionName, tt.expectedSession)
      }
      if wasExisting != tt.expectedExisting {
        t.Errorf("ConnectToProfile() wasExisting = %v, want %v", wasExisting, tt.expectedExisting)
      }
    })
  }
}

func TestCreateWindow(t *testing.T) {
  tests := []struct {
    name        string
    sessionName string
    windowName  string
    mockCmd     func(name string, arg ...string) *exec.Cmd
    expectError bool
  }{
    {
      name:        "create window success",
      sessionName: "test-session",
      windowName:  "server1",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "window created")
      },
      expectError: false,
    },
    {
      name:        "create window failure",
      sessionName: "test-session",
      windowName:  "server1",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("false")
      },
      expectError: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      err := manager.CreateWindow(tt.sessionName, tt.windowName)
      if (err != nil) != tt.expectError {
        t.Errorf("CreateWindow() error = %v, expectError %v", err, tt.expectError)
      }
    })
  }
}

func TestRenameWindow(t *testing.T) {
  tests := []struct {
    name         string
    sessionName  string
    windowNumber string
    newName      string
    mockCmd      func(name string, arg ...string) *exec.Cmd
    expectError  bool
  }{
    {
      name:         "rename window success",
      sessionName:  "test-session",
      windowNumber: "0",
      newName:      "web-server",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "window renamed")
      },
      expectError: false,
    },
    {
      name:         "rename window failure",
      sessionName:  "test-session",
      windowNumber: "0",
      newName:      "web-server",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("false")
      },
      expectError: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      err := manager.RenameWindow(tt.sessionName, tt.windowNumber, tt.newName)
      if (err != nil) != tt.expectError {
        t.Errorf("RenameWindow() error = %v, expectError %v", err, tt.expectError)
      }
    })
  }
}

func TestSendKeysToWindow(t *testing.T) {
  tests := []struct {
    name         string
    windowTarget string
    command      string
    mockCmd      func(name string, arg ...string) *exec.Cmd
    expectError  bool
  }{
    {
      name:         "send keys to window success",
      windowTarget: "test-session:0",
      command:      "ssh user@host",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "keys sent")
      },
      expectError: false,
    },
    {
      name:         "send keys to window failure",
      windowTarget: "test-session:0",
      command:      "ssh user@host",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("false")
      },
      expectError: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      err := manager.SendKeysToWindow(tt.windowTarget, tt.command)
      if (err != nil) != tt.expectError {
        t.Errorf("SendKeysToWindow() error = %v, expectError %v", err, tt.expectError)
      }
    })
  }
}

func TestBuildSSHCommand(t *testing.T) {
  tests := []struct {
    name        string
    server      Server
    expected    string
    expectError bool
  }{
    {
      name: "basic ssh command with key",
      server: &mockServer{
        name:     "web1",
        hostname: "web1.dev.com",
        port:     22,
        username: "dev",
        authType: "key",
        keyPath:  "~/.ssh/id_rsa",
        valid:    true,
      },
      expected:    "ssh -t dev@web1.dev.com -i ~/.ssh/id_rsa -o ServerAliveInterval=60 -o ServerAliveCountMax=3",
      expectError: false,
    },
    {
      name: "ssh command with custom port",
      server: &mockServer{
        name:     "api1",
        hostname: "api1.prod.com",
        port:     2222,
        username: "deploy",
        authType: "key",
        keyPath:  "~/.ssh/deploy_rsa",
        valid:    true,
      },
      expected:    "ssh -t deploy@api1.prod.com -p 2222 -i ~/.ssh/deploy_rsa -o ServerAliveInterval=60 -o ServerAliveCountMax=3",
      expectError: false,
    },
    {
      name: "ssh command with password auth",
      server: &mockServer{
        name:     "legacy1",
        hostname: "legacy1.company.com",
        port:     22,
        username: "admin",
        authType: "password",
        keyPath:  "",
        valid:    true,
      },
      expected:    "ssh -t admin@legacy1.company.com -o ServerAliveInterval=60 -o ServerAliveCountMax=3",
      expectError: false,
    },
    {
      name: "invalid server configuration",
      server: &mockServer{
        name:     "invalid",
        hostname: "",
        port:     22,
        username: "",
        authType: "key",
        keyPath:  "",
        valid:    false,
      },
      expected:    "",
      expectError: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      manager := &Manager{}
      result, err := manager.buildSSHCommand(tt.server)
      if (err != nil) != tt.expectError {
        t.Errorf("buildSSHCommand() error = %v, expectError %v", err, tt.expectError)
        return
      }
      if result != tt.expected {
        t.Errorf("buildSSHCommand() = %v, want %v", result, tt.expected)
      }
    })
  }
}

func TestKillSession(t *testing.T) {
  tests := []struct {
    name        string
    sessionName string
    mockCmd     func(name string, arg ...string) *exec.Cmd
    expectError bool
  }{
    {
      name:        "kill session success",
      sessionName: "test-session",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "session killed")
      },
      expectError: false,
    },
    {
      name:        "kill session failure",
      sessionName: "test-session",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("false")
      },
      expectError: true,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      err := manager.KillSession(tt.sessionName)
      if (err != nil) != tt.expectError {
        t.Errorf("KillSession() error = %v, expectError %v", err, tt.expectError)
      }
    })
  }
}

// Helper function to compare string slices
func stringSliceEqual(a, b []string) bool {
  if len(a) != len(b) {
    return false
  }
  for i, v := range a {
    if v != b[i] {
      return false
    }
  }
  return true
}