package models

import (
	"net/http"
	"strconv"
)

type Machine struct {
	MacAddress string `json:"mac"`
	PortIdx    string `json:"port"`
}

// GetMachine extracts machine information from HTTP headers with optional global MAC address fallback
// X-MAC-Address header takes priority over globalMacAddress
// X-Port header is required
func GetMachine(r *http.Request) Machine {
	return Machine{
		MacAddress: r.Header.Get("X-MAC-Address"),
		PortIdx:    r.Header.Get("X-Port"),
	}
}

// GetMachineWithGlobal extracts machine information from HTTP headers with global MAC address fallback
// X-MAC-Address header takes priority over globalMacAddress
// If X-MAC-Address is not provided, globalMacAddress is used
// X-Port header is required
func GetMachineWithGlobal(r *http.Request, globalMacAddress string) Machine {
	macAddress := r.Header.Get("X-MAC-Address")
	if macAddress == "" {
		macAddress = globalMacAddress
	}

	return Machine{
		MacAddress: macAddress,
		PortIdx:    r.Header.Get("X-Port"),
	}
}

func (m *Machine) GetPort() int {
	port, err := strconv.Atoi(m.PortIdx)
	if err != nil {
		return 0
	}
	return port
}
