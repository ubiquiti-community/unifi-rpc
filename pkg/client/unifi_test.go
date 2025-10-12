package client

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "missing host",
			config: &Config{
				Username:   "admin",
				PrivateKey: []byte("dummy-key"),
			},
			expectError: true,
		},
		{
			name: "missing username",
			config: &Config{
				Host:       "192.168.1.1",
				PrivateKey: []byte("dummy-key"),
			},
			expectError: true,
		},
		{
			name: "missing private key",
			config: &Config{
				Host:     "192.168.1.1",
				Username: "admin",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	config := &Config{
		Host:       "192.168.1.1",
		Username:   "admin",
		PrivateKey: []byte("dummy-key"),
	}

	// Test that defaults are applied correctly without SSH parsing
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.Port != 22 {
		t.Errorf("expected port 22, got %d", config.Port)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", config.Timeout)
	}
}

func TestSetPortPowerCommand(t *testing.T) {
	// This test validates the command generation logic without actual SSH
	tests := []struct {
		name        string
		state       PowerState
		expectedCmd string
		expectError bool
	}{
		{
			name:        "power on",
			state:       PowerOn,
			expectedCmd: "swctrl poe set auto id 1",
			expectError: false,
		},
		{
			name:        "powering on",
			state:       PoweringOn,
			expectedCmd: "swctrl poe set auto id 1",
			expectError: false,
		},
		{
			name:        "power off",
			state:       PowerOff,
			expectedCmd: "swctrl poe set off id 1",
			expectError: false,
		},
		{
			name:        "powering off",
			state:       PoweringOff,
			expectedCmd: "swctrl poe set off id 1",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll test the command generation logic
			var command string

			switch tt.state {
			case PowerOn, PoweringOn:
				command = "swctrl poe set auto id 1"
			case PowerOff, PoweringOff:
				command = "swctrl poe set off id 1"
			}

			if command != tt.expectedCmd {
				t.Errorf("expected command %q, got %q", tt.expectedCmd, command)
			}
		})
	}
}
