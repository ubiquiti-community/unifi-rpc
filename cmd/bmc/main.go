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

// loggingMiddleware logs HTTP requests using slog.
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

// responseWriter wraps http.ResponseWriter to capture the status code.
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
	Long: `UniFi RPC Server provides BMC-style power management for UniFi devices via SSH.

Configuration can be provided via command line flags, environment variables, or config file.
Environment variables are prefixed with UNIFI_RPC_ (e.g., UNIFI_RPC_PORT)

SSH Configuration:
  Switch host: --switch-host or UNIFI_RPC_SWITCH_HOST
  SSH port: --ssh-port or UNIFI_RPC_SSH_PORT (default: 22)
  SSH username: --ssh-username or UNIFI_RPC_SSH_USERNAME (default: root)
  SSH key path: --ssh-key-path or UNIFI_RPC_SSH_KEY_PATH

Device Control:
  X-Port header is required to specify the port number

Example usage:
  unifi-rpc --switch-host=10.0.24.136 --ssh-key-path=~/.ssh/id_rsa
  UNIFI_RPC_SWITCH_HOST=10.0.24.136 UNIFI_RPC_SSH_KEY_PATH=~/.ssh/id_rsa unifi-rpc`,
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

		// Create the BMC service
		svc, err := api.NewBMCService(*cfg)
		if err != nil {
			log.Fatalf("Error creating BMC service: %v", err)
		}

		// Setup routing with standard library
		mux := http.NewServeMux()

		// BMC RPC endpoint (used by bmclib)
		mux.HandleFunc("POST /", svc.RpcHandler)

		// Wrap with logging middleware
		handler := loggingMiddleware(logger)(mux)

		fmt.Printf("Server is running on http://%s:%d\n", cfg.Address, cfg.Port)
		fmt.Printf("Power control via SSH to %s:%d\n", cfg.SwitchHost, cfg.SSHPort)
		fmt.Printf("Port identification via header: X-Port\n")

		logger.Info("starting server",
			slog.String("address", cfg.Address),
			slog.Int("port", cfg.Port),
			slog.String("switch_host", cfg.SwitchHost),
			slog.Int("ssh_port", cfg.SSHPort),
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
