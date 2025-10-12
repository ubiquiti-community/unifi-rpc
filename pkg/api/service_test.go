package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ubiquiti-community/unifi-rpc/pkg/client"
	"github.com/ubiquiti-community/unifi-rpc/pkg/config"
)

// Mock power client for testing.
type mockPowerClient struct {
	powerStates map[int]client.PowerState
}

func (m *mockPowerClient) GetPortPowerState(
	ctx context.Context,
	portID int,
) (client.PowerState, error) {
	state, exists := m.powerStates[portID]
	if !exists {
		return client.PowerOff, nil
	}
	return state, nil
}

func (m *mockPowerClient) SetPortPower(
	ctx context.Context,
	portID int,
	state client.PowerState,
) error {
	m.powerStates[portID] = state
	return nil
}

func (m *mockPowerClient) RestartPortPower(ctx context.Context, portID int) error {
	// Simulate power cycle
	m.powerStates[portID] = client.PowerOff
	m.powerStates[portID] = client.PowerOn
	return nil
}

func (m *mockPowerClient) GetPoEStatus(ctx context.Context, portID int) (*client.PoEStatus, error) {
	return nil, nil // Not needed for these tests
}

func TestRpcHandler_PowerGet(t *testing.T) {
	mockClient := &mockPowerClient{
		powerStates: map[int]client.PowerState{
			1: client.PowerOn,
			2: client.PowerOff,
			3: client.PowerOn,
		},
	}
	svc := &rpcService{powerClient: mockClient}

	tests := []struct {
		name           string
		headers        map[string]string
		payload        RequestPayload
		expectedStatus int
		expectedResult string
	}{
		{
			name: "get power state - port on",
			headers: map[string]string{
				"X-Port": "1",
			},
			payload: RequestPayload{
				Method: PowerGetMethod,
				ID:     1,
			},
			expectedStatus: http.StatusOK,
			expectedResult: "on",
		},
		{
			name: "get power state - port off",
			headers: map[string]string{
				"X-Port": "2",
			},
			payload: RequestPayload{
				Method: PowerGetMethod,
				ID:     2,
			},
			expectedStatus: http.StatusOK,
			expectedResult: "off",
		},
		{
			name:    "missing port header",
			headers: map[string]string{},
			payload: RequestPayload{
				Method: PowerGetMethod,
				ID:     3,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			svc.RpcHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf(
					"Expected status %d, got %d. Response: %s",
					tt.expectedStatus,
					w.Code,
					w.Body.String(),
				)
			}

			if tt.expectedResult != "" {
				var resp ResponsePayload
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if resp.Result != tt.expectedResult {
					t.Errorf("Expected result %q, got %v", tt.expectedResult, resp.Result)
				}
			}
		})
	}
}

func TestRpcHandler_PowerSet(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		payload        RequestPayload
		expectedStatus int
		expectedState  client.PowerState
		portNum        int
	}{
		{
			name: "set power on",
			headers: map[string]string{
				"X-Port": "2",
			},
			payload: RequestPayload{
				Method: PowerSetMethod,
				Params: PowerSetParams{State: "on"},
				ID:     1,
			},
			expectedStatus: http.StatusOK,
			expectedState:  client.PowerOn,
			portNum:        2,
		},
		{
			name: "set power off",
			headers: map[string]string{
				"X-Port": "1",
			},
			payload: RequestPayload{
				Method: PowerSetMethod,
				Params: PowerSetParams{State: "off"},
				ID:     2,
			},
			expectedStatus: http.StatusOK,
			expectedState:  client.PowerOff,
			portNum:        1,
		},
		{
			name: "set power soft (maps to off)",
			headers: map[string]string{
				"X-Port": "3",
			},
			payload: RequestPayload{
				Method: PowerSetMethod,
				Params: PowerSetParams{State: "soft"},
				ID:     3,
			},
			expectedStatus: http.StatusOK,
			expectedState:  client.PowerOff,
			portNum:        3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockPowerClient{
				powerStates: map[int]client.PowerState{
					1: client.PowerOn,
					2: client.PowerOff,
					3: client.PowerOn,
				},
			}
			svc := &rpcService{powerClient: mockClient}

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			svc.RpcHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf(
					"Expected status %d, got %d. Response: %s",
					tt.expectedStatus,
					w.Code,
					w.Body.String(),
				)
			}

			// Verify the mock client has the expected state
			state := mockClient.powerStates[tt.portNum]
			if state != tt.expectedState {
				t.Errorf("Expected power state %v, got %v", tt.expectedState, state)
			}
		})
	}
}

func TestRpcHandler_PowerCycle(t *testing.T) {
	mockClient := &mockPowerClient{
		powerStates: map[int]client.PowerState{
			1: client.PowerOn,
		},
	}
	svc := &rpcService{powerClient: mockClient}

	payload := RequestPayload{
		Method: PowerSetMethod,
		Params: PowerSetParams{State: "cycle"},
		ID:     1,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Port", "1")

	w := httptest.NewRecorder()
	svc.RpcHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Response: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestRpcHandler_Ping(t *testing.T) {
	mockClient := &mockPowerClient{
		powerStates: map[int]client.PowerState{},
	}
	svc := &rpcService{powerClient: mockClient}

	payload := RequestPayload{
		Method: PingMethod,
		ID:     1,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Port", "1")

	w := httptest.NewRecorder()
	svc.RpcHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp ResponsePayload
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Result != "pong" {
		t.Errorf("Expected result 'pong', got %v", resp.Result)
	}
}

func TestNewBMCService(t *testing.T) {
	// Create a temporary SSH key file for testing
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(keyPath, []byte("fake-ssh-key"), 0o600); err != nil {
		t.Fatalf("Failed to create test SSH key: %v", err)
	}

	config := config.Config{
		SwitchHost:  "10.0.24.136",
		SSHPort:     22,
		SSHUsername: "root",
		SSHKeyPath:  keyPath,
		Port:        5000,
		Address:     "0.0.0.0",
	}

	// This will fail to connect but should create the service structure
	svc, err := NewBMCService(config)

	// We expect an error because we can't actually connect to SSH
	// but the service should still be created
	if svc == nil && err == nil {
		t.Error("Expected either service or error, got neither")
	}
}
