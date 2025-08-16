package ssh

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := ClientConfig{
		Hostname: "example.com",
		Port:     22,
		Username: "testuser",
		Timeout:  30 * time.Second,
	}

	client := NewClient(config)
	if client == nil {
		t.Fatal("Expected client to be initialized")
	}

	if client.config.Hostname != "example.com" {
		t.Errorf("Expected hostname 'example.com', got '%s'", client.config.Hostname)
	}
	if client.config.Port != 22 {
		t.Errorf("Expected port 22, got %d", client.config.Port)
	}
}

func TestClientConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ClientConfig{
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing hostname",
			config: ClientConfig{
				Port:     22,
				Username: "user",
				Timeout:  30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "missing username",
			config: ClientConfig{
				Hostname: "example.com",
				Port:     22,
				Timeout:  30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: ClientConfig{
				Hostname: "example.com",
				Port:     0,
				Username: "user",
				Timeout:  30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: ClientConfig{
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				Timeout:  0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ClientConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeyAuthConfig(t *testing.T) {
	tests := []struct {
		name       string
		keyPath    string
		passphrase string
		wantErr    bool
	}{
		{
			name:       "empty key path",
			keyPath:    "",
			passphrase: "",
			wantErr:    true,
		},
		{
			name:       "nonexistent key path",
			keyPath:    "/nonexistent/key/path",
			passphrase: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := NewKeyAuth(tt.keyPath, tt.passphrase)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKeyAuth() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && auth == nil {
				t.Error("Expected auth method to be created")
			}
		})
	}
}

func TestPasswordAuthConfig(t *testing.T) {
	password := "testpassword"
	auth := NewPasswordAuth(password)
	
	if auth == nil {
		t.Fatal("Expected password auth to be created")
	}
}

func TestAgentAuthConfig(t *testing.T) {
	// Test creating agent auth (this will fail if no agent is running, but shouldn't error)
	auth, err := NewAgentAuth()
	
	// Agent auth should be created even if no agent is available
	// The connection attempt will handle agent availability
	if err != nil {
		t.Errorf("NewAgentAuth() should not error during creation, got: %v", err)
	}
	if auth == nil {
		t.Error("Expected agent auth to be created")
	}
}

func TestConnectTimeout(t *testing.T) {
	config := ClientConfig{
		Hostname: "192.0.2.1", // RFC 5737 test address (should timeout)
		Port:     22,
		Username: "test",
		Timeout:  1 * time.Second, // Short timeout for test
	}

	client := NewClient(config)
	
	// Try to connect with password auth (should timeout)
	auth := NewPasswordAuth("test")
	
	start := time.Now()
	err := client.Connect(auth)
	duration := time.Since(start)
	
	if err == nil {
		t.Error("Expected connection to fail due to timeout")
	}
	
	// Should timeout within reasonable time (allowing some buffer)
	if duration > 5*time.Second {
		t.Errorf("Connection took too long: %v", duration)
	}
	
	// Should be disconnected after failed connection
	if client.IsConnected() {
		t.Error("Client should not be connected after failed connection")
	}
}

func TestClientDisconnect(t *testing.T) {
	config := ClientConfig{
		Hostname: "example.com",
		Port:     22,
		Username: "test",
		Timeout:  30 * time.Second,
	}

	client := NewClient(config)
	
	// Should be able to disconnect even if not connected
	err := client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() should not error when not connected, got: %v", err)
	}
	
	if client.IsConnected() {
		t.Error("Client should not be connected after disconnect")
	}
}

func TestExecuteCommand(t *testing.T) {
	config := ClientConfig{
		Hostname: "example.com",
		Port:     22,
		Username: "test",
		Timeout:  30 * time.Second,
	}

	client := NewClient(config)
	
	// Should error when not connected
	output, err := client.ExecuteCommand("echo test")
	if err == nil {
		t.Error("Expected error when executing command without connection")
	}
	if output != "" {
		t.Errorf("Expected empty output when not connected, got: %s", output)
	}
}