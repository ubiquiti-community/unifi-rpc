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
	StatusHandler(w http.ResponseWriter, r *http.Request)
	PowerOnHandler(w http.ResponseWriter, r *http.Request)
	PowerOffHandler(w http.ResponseWriter, r *http.Request)
	PxeBootHandler(w http.ResponseWriter, r *http.Request)
	RebootHandler(w http.ResponseWriter, r *http.Request)
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
		return fmt.Errorf("Missing X-MAC-Address header and no global device MAC address configured")
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
	if state == "on" {
		stateBool = true
	} else if state == "off" {
		stateBool = false
	} else {
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

func (b *rpcService) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if err := validateHeaders(r, b.globalMacAddress); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	machine := models.GetMachineWithGlobal(r, b.globalMacAddress)

	isPoweredOn, err := b.isPoweredOn(r.Context(), machine.MacAddress, machine.PortIdx)
	if err != nil {
		writeError(
			w,
			http.StatusNotFound,
			fmt.Sprintf(
				"error checking power state for MAC Address %s, Port Index %s: %v",
				machine.MacAddress,
				machine.PortIdx,
				err,
			),
		)
		return
	}

	status := struct {
		PoweredOn bool
	}{
		PoweredOn: isPoweredOn,
	}

	fmt.Fprintf(
		w,
		"power state for MAC Address %s, Port Index %s",
		machine.MacAddress,
		machine.PortIdx,
	)

	if err = json.NewEncoder(w).Encode(status); err != nil {
		fmt.Fprintf(
			w,
			"error encoding power state for MAC Address %s, Port Index %s: %v",
			machine.MacAddress,
			machine.PortIdx,
			err,
		)
		return
	}
}

func (b *rpcService) PowerOnHandler(w http.ResponseWriter, r *http.Request) {
	if err := validateHeaders(r, b.globalMacAddress); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	machine := models.GetMachineWithGlobal(r, b.globalMacAddress)

	if err := b.setPower(r.Context(), machine.MacAddress, machine.PortIdx, true); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"error setting power on for MAC Address %s, Port Index %s: %v",
				machine.MacAddress,
				machine.PortIdx,
				err,
			),
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "power on successful"})
}

func (b *rpcService) PowerOffHandler(w http.ResponseWriter, r *http.Request) {
	if err := validateHeaders(r, b.globalMacAddress); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	machine := models.GetMachineWithGlobal(r, b.globalMacAddress)

	if err := b.setPower(r.Context(), machine.MacAddress, machine.PortIdx, false); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"error setting power off for MAC Address %s, Port Index %s: %v",
				machine.MacAddress,
				machine.PortIdx,
				err,
			),
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "power off successful"})
}

func (b *rpcService) PxeBootHandler(w http.ResponseWriter, r *http.Request) {
	if err := validateHeaders(r, b.globalMacAddress); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	machine := models.GetMachineWithGlobal(r, b.globalMacAddress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "pxe boot requested",
		"mac":    machine.MacAddress,
		"port":   machine.PortIdx,
	})
}

func Ptr[T any](v T) *T {
	return &v
}

func (b *rpcService) RebootHandler(w http.ResponseWriter, r *http.Request) {
	if err := validateHeaders(r, b.globalMacAddress); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	machine := models.GetMachineWithGlobal(r, b.globalMacAddress)

	if _, err := b.client.ExecuteCmd(r.Context(), "default", "devmgr", unifi.Cmd{
		Command: "power-cycle",
		Mac:     machine.MacAddress,
		PortIdx: Ptr(machine.GetPort()),
	}); err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"error rebooting device for MAC Address %s, Port Index %s: %v",
				machine.MacAddress,
				machine.PortIdx,
				err,
			),
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reboot successful"})
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
			log.Fatalf(
				"error getting power state for MAC Address %s, Port Index %s: %v",
				machine.MacAddress,
				machine.PortIdx,
				err,
			)
			fmt.Fprintf(
				w,
				"error getting power state for MAC Address %s, Port Index %s: %v",
				machine.MacAddress,
				machine.PortIdx,
				err,
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rp.Result = state
	case PowerSetMethod:
		p, ok := req.Params.(PowerSetParams)
		if !ok {
			log.Fatalf("error asserting params to PowerSetParams")
			fmt.Fprintf(w, "error asserting params to PowerSetParams")
			w.WriteHeader(http.StatusBadRequest)
		}
		state := p.State
		err := b.setPortPower(r.Context(), machine.MacAddress, machine.PortIdx, state)
		if err != nil {
			log.Fatalf(
				"error setting power on for MAC Address %s, Port Index %s: %v",
				machine.MacAddress,
				machine.PortIdx,
				err,
			)
			fmt.Fprintf(
				w,
				"error setting power on for MAC Address %s, Port Index %s: %v",
				machine.MacAddress,
				machine.PortIdx,
				err,
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	case BootDeviceMethod:
		p, ok := req.Params.(BootDeviceParams)
		if !ok {
			log.Fatalf("error asserting params to BootDeviceParams")
			fmt.Fprintf(w, "error asserting params to BootDeviceParams")
			w.WriteHeader(http.StatusBadRequest)
		}
		fmt.Fprintf(
			w,
			"boot device request for MAC Address %s, Port Index %s, Device %s, Persistent %t, EFIBoot %t",
			machine.MacAddress,
			machine.PortIdx,
			p.Device,
			p.Persistent,
			p.EFIBoot,
		)

	case PingMethod:

		rp.Result = "pong"
	default:
		w.WriteHeader(http.StatusNotFound)
	}
	by, _ := json.Marshal(rp)
	w.Write(by)
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
