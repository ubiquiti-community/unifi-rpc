package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/unifi-rpc/pkg/client"
	"github.com/ubiquiti-community/unifi-rpc/pkg/config"
	"github.com/ubiquiti-community/unifi-rpc/pkg/models"
)

type RpcService interface {
	RpcHandler(w http.ResponseWriter, r *http.Request)
}

type rpcService struct {
	client           client.Client
	globalMacAddress string
}

// validateHeaders checks if required headers for machine identification are present
// MAC address is optional if globalMacAddress is configured
func validateHeaders(r *http.Request, globalMacAddress string) error {
	macAddr := r.Header.Get("X-MAC-Address")
	port := r.Header.Get("X-Port")

	if macAddr == "" && globalMacAddress == "" {
		return fmt.Errorf(
			"Missing X-MAC-Address header and no global device MAC address configured",
		)
	}
	if port == "" {
		return fmt.Errorf("Missing X-Port header")
	}
	return nil
}

// writeError writes an error response in JSON format
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (b *rpcService) getPort(
	ctx context.Context,
	macAddress string,
	portIdx string,
) (deviceId string, port unifi.DevicePortOverrides, err error) {
	deviceId = ""

	p, err := strconv.Atoi(portIdx)
	if err != nil {
		err = fmt.Errorf("error getting integer value from port %s: %v", portIdx, err)
		return
	}

	dev, err := b.client.GetDeviceByMAC(ctx, "default", macAddress)
	if err != nil {
		err = fmt.Errorf("error getting device by MAC Address %s: %v", macAddress, err)
		return
	}

	deviceId = dev.ID

	for _, pd := range dev.PortOverrides {
		if pd.PortIDX == p {
			port = pd
			break
		}
	}

	return
}

func (b *rpcService) setPower(
	ctx context.Context,
	macAddress string,
	portIdx string,
	state bool,
) error {
	p, err := strconv.Atoi(portIdx)
	if err != nil {
		return fmt.Errorf("error getting integer value from port %s: %v", portIdx, err)
	}

	dev, err := b.client.GetDeviceByMAC(ctx, "default", macAddress)
	if err != nil {
		return fmt.Errorf("error getting device by MAC Address %s: %v", macAddress, err)
	}

	for i, pd := range dev.PortOverrides {
		if pd.PortIDX == p {
			if state {
				if pd.PoeMode == "auto" {
					return nil
				}
				dev.PortOverrides[i].PoeMode = "auto"
				break
			} else {
				if pd.PoeMode == "off" {
					return nil
				}
				dev.PortOverrides[i].PoeMode = "off"
				break
			}
		}
	}

	_, err = b.client.UpdateDevice(ctx, "default", dev)
	if err != nil {
		return fmt.Errorf("error updating device: %v", err)
	}

	return nil
}

func (b *rpcService) setPortPower(
	ctx context.Context,
	macAddress string,
	portIdx string,
	state string,
) error {
	stateBool := false
	switch state {
	case "on":
		stateBool = true
	case "off":
		stateBool = false
	default:
		return fmt.Errorf("invalid power state %s", state)
	}
	return b.setPower(ctx, macAddress, portIdx, stateBool)
}

func (b *rpcService) isPoweredOn(
	ctx context.Context,
	macAddress string,
	portIdx string,
) (bool, error) {
	_, port, err := b.getPort(ctx, macAddress, portIdx)
	if err != nil {
		fmt.Printf(
			"error setting power on for MAC Address %s, Port Index %s: %v",
			macAddress,
			portIdx,
			err,
		)
		return false, err
	}

	return port.PoeMode == "auto", nil
}

func (b *rpcService) GetPower(
	ctx context.Context,
	macAddress string,
	portIdx string,
) (state string, err error) {
	isPoweredOn, err := b.isPoweredOn(ctx, macAddress, portIdx)
	if err != nil {
		fmt.Printf(
			"error setting power on for MAC Address %s, Port Index %s: %v",
			macAddress,
			portIdx,
			err,
		)
		return
	}

	if isPoweredOn {
		state = "on"
	} else {
		state = "off"
	}

	return
}

// Ptr is a helper function to get a pointer to a value
func Ptr[T any](v T) *T {
	return &v
}

func (b *rpcService) RpcHandler(w http.ResponseWriter, r *http.Request) {
	if err := validateHeaders(r, b.globalMacAddress); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	machine := models.GetMachineWithGlobal(r, b.globalMacAddress)

	req := RequestPayload{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	rp := ResponsePayload{
		ID:   req.ID,
		Host: req.Host,
	}
	switch req.Method {
	case PowerGetMethod:
		state, err := b.GetPower(r.Context(), machine.MacAddress, machine.PortIdx)
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
			if state == "soft" {
				state = "off"
			}
			err := b.setPortPower(r.Context(), machine.MacAddress, machine.PortIdx, state)
			if err != nil {
				rp.Error = &ResponseError{
					Code:    http.StatusInternalServerError,
					Message: fmt.Sprintf("error setting power state: %v", err),
				}
			}

		case "reset", "cycle":
			if _, err := b.client.ExecuteCmd(r.Context(), "default", "devmgr", unifi.Cmd{
				Command: "power-cycle",
				Mac:     machine.MacAddress,
				PortIdx: Ptr(machine.GetPort()),
			}); err != nil {
				rp.Error = &ResponseError{
					Code:    http.StatusInternalServerError,
					Message: fmt.Sprintf("error power cycling device: %v", err),
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

func NewBMCService(cfg config.Config) RpcService {
	return &rpcService{
		client: client.NewClient(
			cfg.Username,
			cfg.Password,
			cfg.APIKey,
			cfg.APIEndpoint,
			cfg.Insecure,
		),
		globalMacAddress: cfg.DeviceMacAddress,
	}
}
