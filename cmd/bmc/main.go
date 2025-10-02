package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ubiquiti-community/unifi-rpc/pkg/api"
	"github.com/ubiquiti-community/unifi-rpc/pkg/config"
)

// loggingMiddleware logs HTTP requests using slog
func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Log the incoming request
			logger.Info("incoming request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.String("mac_address", r.Header.Get("X-MAC-Address")),
				slog.String("port", r.Header.Get("X-Port")),
			)

			// Call the next handler
			next.ServeHTTP(wrapped, r)

			// Log the response
			duration := time.Since(start)
			logger.Info("request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

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

Device Configuration:
  Global device MAC address can be set via --device-mac-address or UNIFI_RPC_DEVICE_MAC_ADDRESS
  X-MAC-Address header will override the global device MAC address when provided
  X-Port header is always required

Example usage:
  unifi-rpc --api-key=your_key --api-endpoint=https://unifi.example.com:8443
  unifi-rpc --device-mac-address=aa:bb:cc:dd:ee:ff --api-key=your_key
  UNIFI_RPC_API_KEY=your_key UNIFI_RPC_DEVICE_MAC_ADDRESS=aa:bb:cc:dd:ee:ff unifi-rpc`,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize slog logger
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		slog.SetDefault(logger)

		// Load configuration using Viper
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading configuration: %v", err)
		}

		// Convert to legacy config format for API compatibility
		legacyConfig := config.Config{
			Username:         cfg.Username,
			Password:         cfg.Password,
			APIKey:           cfg.APIKey,
			APIEndpoint:      cfg.APIEndpoint,
			Insecure:         cfg.Insecure,
			DeviceMacAddress: cfg.DeviceMacAddress,
		}

		svc := api.NewBMCService(legacyConfig)

		// Setup routing with standard library
		mux := http.NewServeMux()

		// BMC RPC endpoint (used by bmclib)
		mux.HandleFunc("POST /", svc.RpcHandler)

		// Wrap with logging middleware
		handler := loggingMiddleware(logger)(mux)

		fmt.Printf("Server is running on http://%s:%d\n", cfg.Address, cfg.Port)
		fmt.Printf("Machine identification via headers: X-MAC-Address, X-Port\n")
		if cfg.DeviceMacAddress != "" {
			fmt.Printf(
				"Global device MAC address: %s (X-MAC-Address header will override)\n",
				cfg.DeviceMacAddress,
			)
		}
		if cfg.APIKey != "" {
			fmt.Printf("Authentication: API Key\n")
		} else {
			fmt.Printf("Authentication: Username/Password\n")
		}
		fmt.Printf("UniFi Controller: %s\n", cfg.APIEndpoint)

		logger.Info("starting server",
			slog.String("address", cfg.Address),
			slog.Int("port", cfg.Port),
			slog.String("unifi_controller", cfg.APIEndpoint),
		)

		err = http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Address, cfg.Port), handler)
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
