package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/paultyng/go-unifi/unifi"

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
	client client.Client
}

func (b *rpcService) getPort(ctx context.Context, macAddress string, portIdx string) (deviceId string, port unifi.DevicePortOverrides, err error) {
	deviceId = ""

	p, err := strconv.Atoi(portIdx)
	if err != nil {
		err = fmt.Errorf("error getting integer value from port %s: %v", portIdx, err)
		return
	}

	dev, err := b.client.GetDeviceByMAC(ctx, macAddress)
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

func (b *rpcService) setPower(ctx context.Context, macAddress string, portIdx string, state bool) error {
	p, err := strconv.Atoi(portIdx)
	if err != nil {
		return fmt.Errorf("error getting integer value from port %s: %v", portIdx, err)
	}

	dev, err := b.client.GetDeviceByMAC(ctx, macAddress)
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

	_, err = b.client.UpdateDevice(ctx, dev)
	if err != nil {
		return fmt.Errorf("error updating device: %v", err)
	}

	return nil
}

func (b *rpcService) setPortPower(ctx context.Context, macAddress string, portIdx string, state string) error {
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

func (b *rpcService) isPoweredOn(ctx context.Context, macAddress string, portIdx string) (bool, error) {
	_, port, err := b.getPort(ctx, macAddress, portIdx)
	if err != nil {
		fmt.Printf("error setting power on for MAC Address %s, Port Index %s: %v", macAddress, portIdx, err)
		return false, err
	}

	return port.PoeMode == "auto", nil
}

func (b *rpcService) GetPower(ctx context.Context, macAddress string, portIdx string) (state string, err error) {
	isPoweredOn, err := b.isPoweredOn(ctx, macAddress, portIdx)
	if err != nil {
		fmt.Printf("error setting power on for MAC Address %s, Port Index %s: %v", macAddress, portIdx, err)
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
	machine := models.GetMachine(r)

	isPoweredOn, err := b.isPoweredOn(r.Context(), machine.MacAddress, machine.PortIdx)
	if err != nil {
		fmt.Fprintf(w, "error checking power state for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
		return
	}

	status := struct {
		PoweredOn bool
	}{
		PoweredOn: isPoweredOn,
	}

	fmt.Fprintf(w, "power state for MAC Address %s, Port Index %s", machine.MacAddress, machine.PortIdx)

	if err = json.NewEncoder(w).Encode(status); err != nil {
		fmt.Fprintf(w, "error encoding power state for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
		return
	}
}

func (b *rpcService) PowerOnHandler(w http.ResponseWriter, r *http.Request) {
	machine := models.GetMachine(r)

	fmt.Fprintf(w, "power on request for MAC Address %s, Port Index %s", machine.MacAddress, machine.PortIdx)
	if err := b.setPower(r.Context(), machine.MacAddress, machine.PortIdx, true); err != nil {
		fmt.Fprintf(w, "error setting power on for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (b *rpcService) PowerOffHandler(w http.ResponseWriter, r *http.Request) {
	machine := models.GetMachine(r)

	fmt.Fprintf(w, "power off request for MAC Address %s, Port Index %s", machine.MacAddress, machine.PortIdx)

	if err := b.setPower(r.Context(), machine.MacAddress, machine.PortIdx, false); err != nil {
		fmt.Fprintf(w, "error setting power off for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (b *rpcService) PxeBootHandler(w http.ResponseWriter, r *http.Request) {
	machine := models.GetMachine(r)
	fmt.Fprintf(w, "pxe boot request for MAC Address %s, Port Index %s", machine.MacAddress, machine.PortIdx)
}

func (b *rpcService) RebootHandler(w http.ResponseWriter, r *http.Request) {
	machine := models.GetMachine(r)
	fmt.Fprintf(w, "reboot request for MAC Address %s, Port Index %s", machine.MacAddress, machine.PortIdx)
}

func (b *rpcService) RpcHandler(w http.ResponseWriter, r *http.Request) {
	machine := models.GetMachine(r)

	req := RequestPayload{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
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
			log.Fatalf("error getting power state for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
			fmt.Fprintf(w, "error getting power state for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
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
			log.Fatalf("error setting power on for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
			fmt.Fprintf(w, "error setting power on for MAC Address %s, Port Index %s: %v", machine.MacAddress, machine.PortIdx, err)
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
		fmt.Fprintf(w, "boot device request for MAC Address %s, Port Index %s, Device %s, Persistent %t, EFIBoot %t", machine.MacAddress, machine.PortIdx, p.Device, p.Persistent, p.EFIBoot)

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
			cfg.APIEndpoint,
			true,
		),
	}
}
