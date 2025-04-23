package models

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Machine struct {
	MacAddress string `json:"mac"`
	PortIdx    string `json:"port"`
}

func GetMachine(r *http.Request) Machine {
	params := mux.Vars(r)

	return Machine{
		MacAddress: params["mac"],
		PortIdx:    params["port"],
	}
}

func (m *Machine) GetPort() int {
	port, err := strconv.Atoi(m.PortIdx)
	if err != nil {
		return 0
	}
	return port
}
