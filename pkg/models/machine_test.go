package models

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// mockMacLookupClient for testing MAC address lookup.
type mockMacLookupClient struct {
	macToPort map[string]int
}

func (m *mockMacLookupClient) GetPortByMacAddress(
	ctx context.Context,
	macAddress string,
) (int, error) {
	if m.macToPort == nil {
		return 0, fmt.Errorf("MAC address %s not found", macAddress)
	}
	port, exists := m.macToPort[macAddress]
	if !exists {
		return 0, fmt.Errorf("MAC address %s not found", macAddress)
	}
	return port, nil
}

func TestGetPort(t *testing.T) {
	tests := []struct {
		name        string
		headers     map[string]string
		expectError bool
		errorMsg    string
		expected    int
	}{
		{
			name: "valid port number",
			headers: map[string]string{
				"X-Port": "5",
			},
			expected: 5,
		},
		{
			name: "valid large port",
			headers: map[string]string{
				"X-Port": "48",
			},
			expected: 48,
		},
		{
			name:        "missing X-Port header",
			headers:     map[string]string{},
			expectError: true,
			errorMsg:    "either X-Port or X-Mac header is required",
		},
		{
			name: "empty X-Port header",
			headers: map[string]string{
				"X-Port": "",
			},
			expectError: true,
			errorMsg:    "either X-Port or X-Mac header is required",
		},
		{
			name: "invalid port - not a number",
			headers: map[string]string{
				"X-Port": "abc",
			},
			expectError: true,
			errorMsg:    "invalid port number",
		},
		{
			name: "invalid port - zero",
			headers: map[string]string{
				"X-Port": "0",
			},
			expectError: true,
			errorMsg:    "port must be positive",
		},
		{
			name: "invalid port - negative",
			headers: map[string]string{
				"X-Port": "-1",
			},
			expectError: true,
			errorMsg:    "port must be positive",
		},
		{
			name: "valid port with whitespace",
			headers: map[string]string{
				"X-Port": " 10 ",
			},
			expected: 10,
		},
		{
			name: "valid MAC address lookup",
			headers: map[string]string{
				"X-Mac": "aa:bb:cc:dd:ee:ff",
			},
			expected: 5,
		},
		{
			name: "invalid MAC address - not found",
			headers: map[string]string{
				"X-Mac": "11:22:33:44:55:66",
			},
			expectError: true,
			errorMsg:    "failed to lookup port by MAC address",
		},
		{
			name: "X-Port takes precedence over X-Mac",
			headers: map[string]string{
				"X-Port": "10",
				"X-Mac":  "aa:bb:cc:dd:ee:ff",
			},
			expected: 10,
		},
	}

	// Create mock client for MAC address lookup
	mockClient := &mockMacLookupClient{
		macToPort: map[string]int{
			"aa:bb:cc:dd:ee:ff": 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/test", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			port, err := GetPort(req, mockClient)

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

			if port.Number != tt.expected {
				t.Errorf("Expected port number %d, got %d", tt.expected, port.Number)
			}
		})
	}
}

// Helper function.
func contains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
