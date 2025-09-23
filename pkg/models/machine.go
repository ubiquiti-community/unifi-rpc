package models

import (
	"net/http"
	"strconv"
)

type Machine struct {
	MacAddress string `json:"mac"`
	PortIdx    string `json:"port"`
}

// GetMachine extracts machine information from HTTP headers
// X-MAC-Address and X-Port headers are used for identification
func GetMachine(r *http.Request) Machine {
	return Machine{
		MacAddress: r.Header.Get("X-MAC-Address"),
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
