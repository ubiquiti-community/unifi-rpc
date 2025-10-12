package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary SSH key file for testing
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(keyPath, []byte("fake-ssh-key"), 0o600); err != nil {
		t.Fatalf("Failed to create test SSH key: %v", err)
	}

	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
		expected    Config
	}{
		{
			name: "valid config with all settings",
			envVars: map[string]string{
				"UNIFI_RPC_SWITCH_HOST":  "10.0.24.136",
				"UNIFI_RPC_SSH_PORT":     "22",
				"UNIFI_RPC_SSH_USERNAME": "root",
				"UNIFI_RPC_SSH_KEY_PATH": keyPath,
				"UNIFI_RPC_PORT":         "8080",
				"UNIFI_RPC_ADDRESS":      "127.0.0.1",
			},
			expected: Config{
				SwitchHost:  "10.0.24.136",
				SSHPort:     22,
				SSHUsername: "root",
				SSHKeyPath:  keyPath,
				Port:        8080,
				Address:     "127.0.0.1",
			},
		},
		{
			name: "valid config with defaults",
			envVars: map[string]string{
				"UNIFI_RPC_SWITCH_HOST":  "192.168.1.1",
				"UNIFI_RPC_SSH_KEY_PATH": keyPath,
			},
			expected: Config{
				SwitchHost:  "192.168.1.1",
				SSHPort:     22,     // default
				SSHUsername: "root", // default
				SSHKeyPath:  keyPath,
				Port:        5000,      // default
				Address:     "0.0.0.0", // default
			},
		},
		{
			name: "missing switch host",
			envVars: map[string]string{
				"UNIFI_RPC_SSH_KEY_PATH": keyPath,
			},
			expectError: true,
			errorMsg:    "switch_host is required",
		},
		{
			name: "missing SSH key path",
			envVars: map[string]string{
				"UNIFI_RPC_SWITCH_HOST": "10.0.24.136",
			},
			expectError: true,
			errorMsg:    "ssh_key_path is required",
		},
		{
			name: "invalid SSH port",
			envVars: map[string]string{
				"UNIFI_RPC_SWITCH_HOST":  "10.0.24.136",
				"UNIFI_RPC_SSH_KEY_PATH": keyPath,
				"UNIFI_RPC_SSH_PORT":     "70000",
			},
			expectError: true,
			errorMsg:    "ssh_port must be between",
		},
		{
			name: "invalid server port",
			envVars: map[string]string{
				"UNIFI_RPC_SWITCH_HOST":  "10.0.24.136",
				"UNIFI_RPC_SSH_KEY_PATH": keyPath,
				"UNIFI_RPC_PORT":         "0",
			},
			expectError: true,
			errorMsg:    "port must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper instance for each test
			viper.Reset()

			// Clear environment variables first
			clearEnvVars()

			// Set test environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			// Initialize configuration after setting env vars
			InitConfig()

			config, err := LoadConfig()

			// Cleanup after test
			clearEnvVars()

			if tt.expectError {
				if err == nil {
					t.Error("Expected an error, got nil")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got: %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config.SwitchHost != tt.expected.SwitchHost {
				t.Errorf(
					"Expected SwitchHost %q, got %q",
					tt.expected.SwitchHost,
					config.SwitchHost,
				)
			}
			if config.SSHPort != tt.expected.SSHPort {
				t.Errorf("Expected SSHPort %d, got %d", tt.expected.SSHPort, config.SSHPort)
			}
			if config.SSHUsername != tt.expected.SSHUsername {
				t.Errorf(
					"Expected SSHUsername %q, got %q",
					tt.expected.SSHUsername,
					config.SSHUsername,
				)
			}
			if config.SSHKeyPath != tt.expected.SSHKeyPath {
				t.Errorf(
					"Expected SSHKeyPath %q, got %q",
					tt.expected.SSHKeyPath,
					config.SSHKeyPath,
				)
			}
			if config.Port != tt.expected.Port {
				t.Errorf("Expected Port %d, got %d", tt.expected.Port, config.Port)
			}
			if config.Address != tt.expected.Address {
				t.Errorf("Expected Address %q, got %q", tt.expected.Address, config.Address)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	// Create a temporary SSH key file for testing
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(keyPath, []byte("fake-ssh-key"), 0o600); err != nil {
		t.Fatalf("Failed to create test SSH key: %v", err)
	}

	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				SwitchHost:  "10.0.24.136",
				SSHPort:     22,
				SSHUsername: "root",
				SSHKeyPath:  keyPath,
				Port:        5000,
				Address:     "0.0.0.0",
			},
		},
		{
			name: "missing switch host",
			config: Config{
				SSHPort:     22,
				SSHUsername: "root",
				SSHKeyPath:  keyPath,
				Port:        5000,
			},
			expectError: true,
			errorMsg:    "switch_host is required",
		},
		{
			name: "missing SSH key path",
			config: Config{
				SwitchHost:  "10.0.24.136",
				SSHPort:     22,
				SSHUsername: "root",
				Port:        5000,
			},
			expectError: true,
			errorMsg:    "ssh_key_path is required",
		},
		{
			name: "invalid SSH port - too high",
			config: Config{
				SwitchHost:  "10.0.24.136",
				SSHPort:     70000,
				SSHUsername: "root",
				SSHKeyPath:  keyPath,
				Port:        5000,
			},
			expectError: true,
			errorMsg:    "ssh_port must be between",
		},
		{
			name: "invalid SSH port - too low",
			config: Config{
				SwitchHost:  "10.0.24.136",
				SSHPort:     0,
				SSHUsername: "root",
				SSHKeyPath:  keyPath,
				Port:        5000,
			},
			expectError: true,
			errorMsg:    "ssh_port must be between",
		},
		{
			name: "invalid server port - too high",
			config: Config{
				SwitchHost:  "10.0.24.136",
				SSHPort:     22,
				SSHUsername: "root",
				SSHKeyPath:  keyPath,
				Port:        70000,
			},
			expectError: true,
			errorMsg:    "port must be between",
		},
		{
			name: "invalid server port - too low",
			config: Config{
				SwitchHost:  "10.0.24.136",
				SSHPort:     22,
				SSHUsername: "root",
				SSHKeyPath:  keyPath,
				Port:        0,
			},
			expectError: true,
			errorMsg:    "port must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected an error, got nil")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got: %v", tt.errorMsg, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Helper functions.
func clearEnvVars() {
	envVars := []string{
		"UNIFI_RPC_PORT",
		"UNIFI_RPC_ADDRESS",
		"UNIFI_RPC_SWITCH_HOST",
		"UNIFI_RPC_SSH_PORT",
		"UNIFI_RPC_SSH_USERNAME",
		"UNIFI_RPC_SSH_KEY_PATH",
	}

	for _, env := range envVars {
		_ = os.Unsetenv(env)
	}
}

func contains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
