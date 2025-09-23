package models

import (
	"net/http"
	"testing"
)

func TestGetMachine(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected Machine
	}{
		{
			name: "valid headers",
			headers: map[string]string{
				"X-MAC-Address": "aa:bb:cc:dd:ee:ff",
				"X-Port":        "1",
			},
			expected: Machine{
				MacAddress: "aa:bb:cc:dd:ee:ff",
				PortIdx:    "1",
			},
		},
		{
			name:    "missing headers",
			headers: map[string]string{},
			expected: Machine{
				MacAddress: "",
				PortIdx:    "",
			},
		},
		{
			name: "partial headers - only MAC",
			headers: map[string]string{
				"X-MAC-Address": "11:22:33:44:55:66",
			},
			expected: Machine{
				MacAddress: "11:22:33:44:55:66",
				PortIdx:    "",
			},
		},
		{
			name: "partial headers - only port",
			headers: map[string]string{
				"X-Port": "8",
			},
			expected: Machine{
				MacAddress: "",
				PortIdx:    "8",
			},
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

			machine := GetMachine(req)

			if machine.MacAddress != tt.expected.MacAddress {
				t.Errorf("Expected MacAddress %q, got %q", tt.expected.MacAddress, machine.MacAddress)
			}
			if machine.PortIdx != tt.expected.PortIdx {
				t.Errorf("Expected PortIdx %q, got %q", tt.expected.PortIdx, machine.PortIdx)
			}
		})
	}
}

func TestMachine_GetPort(t *testing.T) {
	tests := []struct {
		name     string
		machine  Machine
		expected int
	}{
		{
			name: "valid port number",
			machine: Machine{
				MacAddress: "aa:bb:cc:dd:ee:ff",
				PortIdx:    "5",
			},
			expected: 5,
		},
		{
			name: "zero port",
			machine: Machine{
				MacAddress: "aa:bb:cc:dd:ee:ff",
				PortIdx:    "0",
			},
			expected: 0,
		},
		{
			name: "invalid port - not a number",
			machine: Machine{
				MacAddress: "aa:bb:cc:dd:ee:ff",
				PortIdx:    "abc",
			},
			expected: 0,
		},
		{
			name: "empty port",
			machine: Machine{
				MacAddress: "aa:bb:cc:dd:ee:ff",
				PortIdx:    "",
			},
			expected: 0,
		},
		{
			name: "negative port",
			machine: Machine{
				MacAddress: "aa:bb:cc:dd:ee:ff",
				PortIdx:    "-1",
			},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := tt.machine.GetPort()
			if port != tt.expected {
				t.Errorf("Expected port %d, got %d", tt.expected, port)
			}
		})
	}
}
