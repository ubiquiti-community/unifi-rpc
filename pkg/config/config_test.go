package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		errorMsg    string
		expected    Config
	}{
		{
			name: "valid config with API key",
			envVars: map[string]string{
				"UNIFI_RPC_API_KEY":             "test-api-key",
				"UNIFI_RPC_API_ENDPOINT":        "https://unifi.example.com:8443",
				"UNIFI_RPC_PORT":                "8080",
				"UNIFI_RPC_ADDRESS":             "127.0.0.1",
				"UNIFI_RPC_INSECURE":            "false",
				"UNIFI_RPC_DEVICE_MAC_ADDRESS":  "aa:bb:cc:dd:ee:ff",
			},
			expected: Config{
				APIKey:           "test-api-key",
				APIEndpoint:      "https://unifi.example.com:8443",
				Port:             8080,
				Address:          "127.0.0.1",
				Insecure:         false,
				DeviceMacAddress: "aa:bb:cc:dd:ee:ff",
			},
		},
		{
			name: "valid config with username/password",
			envVars: map[string]string{
				"UNIFI_RPC_USERNAME":     "admin",
				"UNIFI_RPC_PASSWORD":     "secret",
				"UNIFI_RPC_API_ENDPOINT": "https://unifi.example.com:8443",
			},
			expected: Config{
				Username:         "admin",
				Password:         "secret",
				APIEndpoint:      "https://unifi.example.com:8443",
				Port:             5000,      // default
				Address:          "0.0.0.0", // default
				Insecure:         true,      // default
				DeviceMacAddress: "",        // default
			},
		},
		{
			name: "default config with minimal settings",
			envVars: map[string]string{
				"UNIFI_RPC_API_KEY": "test-key",
			},
			expected: Config{
				APIKey:           "test-key",
				APIEndpoint:      "https://10.0.0.1", // default
				Port:             5000,               // default
				Address:          "0.0.0.0",          // default
				Insecure:         true,               // default
				DeviceMacAddress: "",                 // default
			},
		},
		{
			name: "missing authentication",
			envVars: map[string]string{
				"UNIFI_RPC_API_ENDPOINT": "https://unifi.example.com:8443",
			},
			expectError: true,
			errorMsg:    "authentication required",
		},
		{
			name: "invalid port",
			envVars: map[string]string{
				"UNIFI_RPC_API_KEY": "test-key",
				"UNIFI_RPC_PORT":    "70000",
			},
			expectError: true,
			errorMsg:    "port must be between",
		},
		{
			name: "config with device MAC address only",
			envVars: map[string]string{
				"UNIFI_RPC_DEVICE_MAC_ADDRESS": "aa:bb:cc:dd:ee:ff",
				"UNIFI_RPC_API_KEY":            "test-key",
			},
			expected: Config{
				APIKey:           "test-key",
				APIEndpoint:      "https://10.0.0.1",     // default
				Port:             5000,                   // default
				Address:          "0.0.0.0",              // default
				Insecure:         true,                   // default
				DeviceMacAddress: "aa:bb:cc:dd:ee:ff",
			},
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
				os.Setenv(key, value)
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

			if config.Username != tt.expected.Username {
				t.Errorf("Expected Username %q, got %q", tt.expected.Username, config.Username)
			}
			if config.Password != tt.expected.Password {
				t.Errorf("Expected Password %q, got %q", tt.expected.Password, config.Password)
			}
			if config.APIKey != tt.expected.APIKey {
				t.Errorf("Expected APIKey %q, got %q", tt.expected.APIKey, config.APIKey)
			}
			if config.APIEndpoint != tt.expected.APIEndpoint {
				t.Errorf(
					"Expected APIEndpoint %q, got %q",
					tt.expected.APIEndpoint,
					config.APIEndpoint,
				)
			}
			if config.Port != tt.expected.Port {
				t.Errorf("Expected Port %d, got %d", tt.expected.Port, config.Port)
			}
			if config.Address != tt.expected.Address {
				t.Errorf("Expected Address %q, got %q", tt.expected.Address, config.Address)
			}
			if config.Insecure != tt.expected.Insecure {
				t.Errorf("Expected Insecure %v, got %v", tt.expected.Insecure, config.Insecure)
			}
			if config.DeviceMacAddress != tt.expected.DeviceMacAddress {
				t.Errorf("Expected DeviceMacAddress %q, got %q", tt.expected.DeviceMacAddress, config.DeviceMacAddress)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with API key",
			config: Config{
				APIKey:      "test-key",
				APIEndpoint: "https://unifi.example.com:8443",
				Port:        5000,
			},
		},
		{
			name: "valid config with username/password",
			config: Config{
				Username:    "admin",
				Password:    "secret",
				APIEndpoint: "https://unifi.example.com:8443",
				Port:        5000,
			},
		},
		{
			name: "missing authentication",
			config: Config{
				APIEndpoint: "https://unifi.example.com:8443",
				Port:        5000,
			},
			expectError: true,
			errorMsg:    "authentication required",
		},
		{
			name: "missing API endpoint",
			config: Config{
				APIKey: "test-key",
				Port:   5000,
			},
			expectError: true,
			errorMsg:    "API_ENDPOINT is required",
		},
		{
			name: "invalid port - too high",
			config: Config{
				APIKey:      "test-key",
				APIEndpoint: "https://unifi.example.com:8443",
				Port:        70000,
			},
			expectError: true,
			errorMsg:    "port must be between",
		},
		{
			name: "invalid port - too low",
			config: Config{
				APIKey:      "test-key",
				APIEndpoint: "https://unifi.example.com:8443",
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

// Helper functions
func clearEnvVars() {
	envVars := []string{
		"UNIFI_RPC_PORT",
		"UNIFI_RPC_ADDRESS",
		"UNIFI_RPC_USERNAME",
		"UNIFI_RPC_PASSWORD",
		"UNIFI_RPC_API_KEY",
		"UNIFI_RPC_API_ENDPOINT",
		"UNIFI_RPC_INSECURE",
		"UNIFI_RPC_DEVICE_MAC_ADDRESS",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
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
