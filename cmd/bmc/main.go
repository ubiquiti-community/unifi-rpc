package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ubiquiti-community/unifi-rpc/pkg/api"
	"github.com/ubiquiti-community/unifi-rpc/pkg/config"
)

var (
	port     int
	filePath string
	address  string
	cfg      config.Config
)

func main() {
	flag.IntVar(&port, "p", 5000, "port to listen on")
	flag.StringVar(&address, "a", "0.0.0.0", "address to listen on")
	flag.StringVar(&filePath, "c", "config.yaml", "configuration yaml file")
	flag.Parse()

	cfg, err := config.GetConfig(filePath)
	if err != nil {
		log.Fatalf("error reading YAML file: %v", err)
	}

	svc := api.NewBMCService(cfg)

	r := mux.NewRouter()

	r.HandleFunc("/device/{mac}/port/{port}/rpc", svc.RpcHandler).Methods("POST")

	// Methods for Sidero Omni Metal API
	r.HandleFunc("/device/{mac}/port/{port}/status", svc.StatusHandler).Methods("GET")
	r.HandleFunc("/device/{mac}/port/{port}/poweron", svc.PowerOnHandler).Methods("POST")
	r.HandleFunc("/device/{mac}/port/{port}/poweroff", svc.PowerOffHandler).Methods("POST")
	r.HandleFunc("/device/{mac}/port/{port}/reboot", svc.RebootHandler).Methods("POST")
	r.HandleFunc("/device/{mac}/port/{port}/pxeboot", svc.PxeBootHandler).Methods("POST")

	fmt.Printf("Server is running on http://%s:%d", address, port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", address, port), nil)
	if err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}
