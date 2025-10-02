package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/unifi-rpc/pkg/config"
)

// Mock client for testing
type mockClient struct {
	devices     map[string]*unifi.Device
	powerStates map[string]bool
}

func (m *mockClient) GetDeviceByMAC(ctx context.Context, site, mac string) (*unifi.Device, error) {
	device, exists := m.devices[mac]
	if !exists {
		return nil, &unifi.NotFoundError{}
	}
	return device, nil
}

func (m *mockClient) UpdateDevice(
	ctx context.Context,
	site string,
	d *unifi.Device,
) (*unifi.Device, error) {
	m.devices[d.MAC] = d
	return d, nil
}

func (m *mockClient) ExecuteCmd(
	ctx context.Context,
	site string,
	mgr string,
	cmd unifi.Cmd,
) (any, error) {
	m.powerStates[cmd.Mac] = !m.powerStates[cmd.Mac] // Toggle power state
	return nil, nil
}

func newTestService() *rpcService {
	return &rpcService{
		client: &mockClient{
			devices: map[string]*unifi.Device{
				"aa:bb:cc:dd:ee:ff": {
					MAC:  "aa:bb:cc:dd:ee:ff",
					Name: "Test Switch",
					Type: "usw",
					PortOverrides: []unifi.DevicePortOverrides{
						{
							PortIDX: 1,
							Name:    "Port 1",
							PoeMode: "auto",
						},
						{
							PortIDX: 2,
							Name:    "Port 2",
							PoeMode: "off",
						},
					},
				},
			},
			powerStates: map[string]bool{
				"aa:bb:cc:dd:ee:ff": true,
			},
		},
	}
}

func TestHeaderValidation(t *testing.T) {
	svc := newTestService()

	tests := []struct {
		name           string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name: "valid headers",
			headers: map[string]string{
				"X-MAC-Address": "aa:bb:cc:dd:ee:ff",
				"X-Port":        "1",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing MAC header",
			headers: map[string]string{
				"X-Port": "1",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing Port header",
			headers: map[string]string{
				"X-MAC-Address": "aa:bb:cc:dd:ee:ff",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "no headers",
			headers:        map[string]string{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/status", nil)
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
		})
	}
}

func TestNewBMCService(t *testing.T) {
	// Test basic service instantiation with minimal config
	// We can't test the full initialization without network access,
	// so we just verify that the constructor can be called
	config := config.Config{
		APIEndpoint: "https://localhost:8443",
		Username:    "test",
		Password:    "test",
		Insecure:    true,
	}

	// Create a defer to catch any panics during initialization
	defer func() {
		if r := recover(); r != nil {
			// Only fail if it's not a network-related error
			if err, ok := r.(error); ok {
				errMsg := err.Error()
				if strings.Contains(errMsg, "dial tcp") ||
					strings.Contains(errMsg, "no such host") ||
					strings.Contains(errMsg, "connection refused") ||
					strings.Contains(errMsg, "connect: connection refused") {
					// Network errors are expected in unit tests
					return
				}
			}
			// For string panics, check if they contain network-related messages
			if errStr, ok := r.(string); ok {
				if strings.Contains(errStr, "dial tcp") ||
					strings.Contains(errStr, "no such host") ||
					strings.Contains(errStr, "connection refused") ||
					strings.Contains(errStr, "connect: connection refused") {
					// Network errors are expected in unit tests
					return
				}
			}
			t.Errorf("NewBMCService panicked with non-network error: %v", r)
		}
	}()

	svc := NewBMCService(config)
	if svc == nil {
		t.Error("Expected non-nil service")
	}
}
