package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/ubiquiti-community/unifi-rpc/pkg/api"
	"github.com/ubiquiti-community/unifi-rpc/pkg/config"
)

var rootCmd = &cobra.Command{
	Use:   "unifi-rpc",
	Short: "UniFi RPC Server - BMC-style power management for UniFi devices",
	Long: `UniFi RPC Server provides BMC-style power management for UniFi devices.

Configuration can be provided via command line flags, environment variables, or config file.
Environment variables are prefixed with UNIFI_RPC_ (e.g., UNIFI_RPC_PORT)

Authentication:
  Use either API key (recommended) or username/password
  API key: --api-key or UNIFI_RPC_API_KEY
  Username/Password: --username/--password or UNIFI_RPC_USERNAME/UNIFI_RPC_PASSWORD

Example usage:
  unifi-rpc --api-key=your_key --api-endpoint=https://unifi.example.com:8443
  UNIFI_RPC_API_KEY=your_key unifi-rpc`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration using Viper
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading configuration: %v", err)
		}

		// Convert to legacy config format for API compatibility
		legacyConfig := config.Config{
			Username:    cfg.Username,
			Password:    cfg.Password,
			APIKey:      cfg.APIKey,
			APIEndpoint: cfg.APIEndpoint,
			Insecure:    cfg.Insecure,
		}

		svc := api.NewBMCService(legacyConfig)

		// Setup routing with standard library - header-based endpoints
		mux := http.NewServeMux()

		// BMC RPC endpoint (used by bmclib)
		mux.HandleFunc("POST /rpc", svc.RpcHandler)

		// Sidero Omni Metal API compatible endpoints
		mux.HandleFunc("GET /status", svc.StatusHandler)
		mux.HandleFunc("POST /poweron", svc.PowerOnHandler)
		mux.HandleFunc("POST /poweroff", svc.PowerOffHandler)
		mux.HandleFunc("POST /reboot", svc.RebootHandler)
		mux.HandleFunc("POST /pxeboot", svc.PxeBootHandler)

		fmt.Printf("Server is running on http://%s:%d\n", cfg.Address, cfg.Port)
		fmt.Printf("Machine identification via headers: X-MAC-Address, X-Port\n")
		if cfg.APIKey != "" {
			fmt.Printf("Authentication: API Key\n")
		} else {
			fmt.Printf("Authentication: Username/Password\n")
		}
		fmt.Printf("UniFi Controller: %s\n", cfg.APIEndpoint)

		err = http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Address, cfg.Port), mux)
		if err != nil {
			log.Fatalf("error starting server: %v", err)
		}
	},
}

func init() {
	cobra.OnInitialize(config.InitConfig)

	// Initialize flags
	config.InitFlags(rootCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
