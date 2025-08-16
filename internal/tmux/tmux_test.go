package tmux

import (
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
    name         string
    serverName   string
    sshCommand   string
    mockCmd      func(name string, arg ...string) *exec.Cmd
    expectError  bool
    expectedSession string
  }{
    {
      name:       "connect to new server",
      serverName: "production-api",
      sshCommand: "ssh deploy@api.prod.company.com",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "connected")
      },
      expectError:     false,
      expectedSession: "production-api",
    },
    {
      name:       "connect with session conflict",
      serverName: "production-api",
      sshCommand: "ssh deploy@api.prod.company.com",
      mockCmd: func(name string, arg ...string) *exec.Cmd {
        return exec.Command("echo", "connected")
      },
      expectError:     false,
      expectedSession: "production-api-1",
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      original := execCommand
      defer func() { execCommand = original }()
      execCommand = tt.mockCmd

      manager := &Manager{}
      if tt.name == "connect with session conflict" {
        manager.existingSessions = []string{"production-api"}
      }

      sessionName, err := manager.ConnectToServer(tt.serverName, tt.sshCommand)
      if (err != nil) != tt.expectError {
        t.Errorf("ConnectToServer() error = %v, expectError %v", err, tt.expectError)
        return
      }
      if sessionName != tt.expectedSession {
        t.Errorf("ConnectToServer() sessionName = %v, want %v", sessionName, tt.expectedSession)
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