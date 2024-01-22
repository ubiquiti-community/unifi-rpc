package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/paultyng/go-unifi/unifi"
	"github.com/ubiquiti-community/unifi-rpc/pkg/config"
)

type BMCService interface {
	RPCHandler(w http.ResponseWriter, r *http.Request)
}

type bmcService struct {
	client *lazyClient
}

func (b *bmcService) getPort(ctx context.Context, macAddress string, portIdx string) (deviceId string, port unifi.DevicePortOverrides, err error) {
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

func (b *bmcService) setPortPower(ctx context.Context, macAddress string, portIdx string, state string) error {
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
			switch state {
			case "on":
				if pd.PoeMode == "auto" {
					return nil
				}
				dev.PortOverrides[i].PoeMode = "auto"
				break
			case "off":
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

func (b *bmcService) GetPower(ctx context.Context, macAddress string, portIdx string) (state string, err error) {
	_, port, err := b.getPort(ctx, macAddress, portIdx)
	if err != nil {
		fmt.Printf("error setting power on for MAC Address %s, Port Index %s: %v", macAddress, portIdx, err)
		return
	}

	mode := port.PoeMode

	if mode == "auto" {
		state = "on"
	} else if mode == "off" {
		state = "off"
	}

	return
}

func getMachine(r *http.Request) Machine {
	params := mux.Vars(r)

	return Machine{
		MacAddress: params["mac"],
		PortIdx:    params["port"],
	}
}

func (b *bmcService) RPCHandler(w http.ResponseWriter, r *http.Request) {
	req := RequestPayload{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	machine := getMachine(r)

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

func NewBMCService(cfg config.Config) BMCService {
	return &bmcService{
		client: &lazyClient{
			user:     cfg.Username,
			pass:     cfg.Password,
			baseURL:  cfg.APIEndpoint,
			insecure: true,
		},
	}
}
