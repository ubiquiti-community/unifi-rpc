package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ubiquiti-community/unifi-rpc/pkg/client"
	"github.com/ubiquiti-community/unifi-rpc/pkg/config"
	"github.com/ubiquiti-community/unifi-rpc/pkg/models"
	"golang.org/x/crypto/ssh"
)

// PowerClient interface for power management operations.
type PowerClient interface {
	GetPortPowerState(ctx context.Context, portID int) (client.PowerState, error)
	SetPortPower(ctx context.Context, portID int, state client.PowerState) error
	RestartPortPower(ctx context.Context, portID int) error
	GetPoEStatus(ctx context.Context, portID int) (*client.PoEStatus, error)
}

type RpcService interface {
	RpcHandler(w http.ResponseWriter, r *http.Request)
}

type rpcService struct {
	powerClient PowerClient
}

// writeError writes an error response in JSON format.
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(map[string]string{"error": message})
	if err != nil {
		log.Printf("error writing error response: %v", err)
	}
}

// getPowerState gets the current power state of a port.
func (s *rpcService) getPowerState(ctx context.Context, portNum int) (string, error) {
	state, err := s.powerClient.GetPortPowerState(ctx, portNum)
	if err != nil {
		return "", fmt.Errorf("failed to get power state: %w", err)
	}

	switch state {
	case client.PowerOn:
		return "on", nil
	case client.PowerOff:
		return "off", nil
	default:
		return "unknown", nil
	}
}

// setPowerState sets the power state of a port.
func (s *rpcService) setPowerState(ctx context.Context, portNum int, state string) error {
	var powerState client.PowerState

	switch state {
	case "on":
		powerState = client.PowerOn
	case "off", "soft":
		powerState = client.PowerOff
	default:
		return fmt.Errorf("invalid power state: %s", state)
	}

	if err := s.powerClient.SetPortPower(ctx, portNum, powerState); err != nil {
		return fmt.Errorf("failed to set power state: %w", err)
	}

	return nil
}

// restartPort performs a power cycle on a port.
func (s *rpcService) restartPort(ctx context.Context, portNum int) error {
	if err := s.powerClient.RestartPortPower(ctx, portNum); err != nil {
		return fmt.Errorf("failed to restart port: %w", err)
	}
	return nil
}

func (s *rpcService) RpcHandler(w http.ResponseWriter, r *http.Request) {
	// Extract port from headers
	port, err := models.GetPort(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse request payload
	req := RequestPayload{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Prepare response
	rp := ResponsePayload{
		ID:   req.ID,
		Host: req.Host,
	}

	// Handle different RPC methods
	switch req.Method {
	case PowerGetMethod:
		state, err := s.getPowerState(r.Context(), port.Number)
		if err != nil {
			rp.Error = &ResponseError{
				Code:    http.StatusInternalServerError,
				Message: fmt.Sprintf("error getting power state: %v", err),
			}
		} else {
			rp.Result = state
		}

	case PowerSetMethod:
		// Parse params from the request
		paramsJSON, err := json.Marshal(req.Params)
		if err != nil {
			rp.Error = &ResponseError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("error marshaling params: %v", err),
			}
			break
		}

		var p PowerSetParams
		if err := json.Unmarshal(paramsJSON, &p); err != nil {
			rp.Error = &ResponseError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("error parsing PowerSetParams: %v", err),
			}
			break
		}

		state := p.State
		switch state {
		case "on", "off", "soft":
			if err := s.setPowerState(r.Context(), port.Number, state); err != nil {
				rp.Error = &ResponseError{
					Code:    http.StatusInternalServerError,
					Message: fmt.Sprintf("error setting power state: %v", err),
				}
			}

		case "reset", "cycle":
			if err := s.restartPort(r.Context(), port.Number); err != nil {
				rp.Error = &ResponseError{
					Code:    http.StatusInternalServerError,
					Message: fmt.Sprintf("error power cycling port: %v", err),
				}
			}

		default:
			rp.Error = &ResponseError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("invalid power state: %s", state),
			}
		}

	case BootDeviceMethod:
		// Parse params from the request
		paramsJSON, err := json.Marshal(req.Params)
		if err != nil {
			rp.Error = &ResponseError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("error marshaling params: %v", err),
			}
			break
		}

		var p BootDeviceParams
		if err := json.Unmarshal(paramsJSON, &p); err != nil {
			rp.Error = &ResponseError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("error parsing BootDeviceParams: %v", err),
			}
			break
		}

		// Boot device method is acknowledged but not implemented for UniFi
		rp.Result = map[string]any{
			"acknowledged": true,
			"device":       p.Device,
			"persistent":   p.Persistent,
			"efiBoot":      p.EFIBoot,
			"message":      "boot device setting not supported for UniFi devices",
		}

	case PingMethod:
		rp.Result = "pong"

	default:
		rp.Error = &ResponseError{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("unknown method: %s", req.Method),
		}
	}

	// Set appropriate HTTP status code
	w.Header().Set("Content-Type", "application/json")
	if rp.Error != nil {
		w.WriteHeader(rp.Error.Code)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	// Write response
	if err := json.NewEncoder(w).Encode(rp); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

// NewBMCService creates a new RPC service with SSH-based power management.
func NewBMCService(cfg config.Config) (RpcService, error) {
	// Read SSH private key from file
	privateKey, err := os.ReadFile(cfg.SSHKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH private key: %w", err)
	}

	// Create power client configuration
	powerConfig := &client.Config{
		Host:            cfg.SwitchHost,
		Port:            cfg.SSHPort,
		Username:        cfg.SSHUsername,
		PrivateKey:      privateKey,
		Timeout:         30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Create the power client
	powerClient, err := client.NewClient(powerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create power client: %w", err)
	}

	return &rpcService{
		powerClient: powerClient,
	}, nil
}
