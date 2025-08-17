package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ParseSSHConfig parses an SSH config file and extracts server configurations
func ParseSSHConfig(configPath string) ([]Server, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSH config file: %w", err)
	}
	defer file.Close()

	var servers []Server
	var currentHost *Server
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Split line into parts
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		
		keyword := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")
		
		switch keyword {
		case "host":
			// Save previous host if it was complete
			if currentHost != nil && isValidServer(currentHost) {
				servers = append(servers, *currentHost)
			}
			
			// Skip wildcard hosts
			if strings.Contains(value, "*") || strings.Contains(value, "?") {
				currentHost = nil
				continue
			}
			
			// Start new host configuration
			currentHost = &Server{
				Name: value,
				Port: 22, // default SSH port
			}
			
		case "hostname":
			if currentHost != nil {
				currentHost.Hostname = value
			}
			
		case "user":
			if currentHost != nil {
				currentHost.Username = value
			}
			
		case "port":
			if currentHost != nil {
				if port, err := strconv.Atoi(value); err == nil {
					currentHost.Port = port
				}
			}
			
		case "identityfile":
			if currentHost != nil {
				currentHost.KeyPath = value
				currentHost.AuthType = "key"
			}
		}
	}
	
	// Don't forget the last host
	if currentHost != nil && isValidServer(currentHost) {
		servers = append(servers, *currentHost)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SSH config file: %w", err)
	}
	
	// Set default auth type for servers without identity files
	for i := range servers {
		if servers[i].AuthType == "" {
			servers[i].AuthType = "password"
		}
	}
	
	return servers, nil
}

// isValidServer checks if a server configuration has the minimum required fields
func isValidServer(server *Server) bool {
	return server.Name != "" && 
		   server.Hostname != "" && 
		   server.Username != ""
}

// DefaultSSHConfigPath returns the default SSH config file path
func DefaultSSHConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	return homeDir + "/.ssh/config", nil
}