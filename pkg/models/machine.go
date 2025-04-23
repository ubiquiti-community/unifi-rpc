package models

import (
	"net/http"

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
