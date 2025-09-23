package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	// Server configuration
	Port    int    `mapstructure:"port"`
	Address string `mapstructure:"address"`

	// UniFi Controller authentication (choose one)
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	APIKey   string `mapstructure:"api_key"`

	// UniFi Controller connection
	APIEndpoint string `mapstructure:"api_endpoint"`
	Insecure    bool   `mapstructure:"insecure"`

	// Device configuration
	DeviceMacAddress string `mapstructure:"device_mac_address"`
}

var (
	cfgFile string
	cfg     *Config
)

// InitConfig initializes viper configuration
func InitConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name "unifi-rpc" (without extension)
		viper.AddConfigPath("$HOME")
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("unifi-rpc")
	}

	// Enable reading from environment variables
	viper.SetEnvPrefix("UNIFI_RPC")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("port", 5000)
	viper.SetDefault("address", "0.0.0.0")
	viper.SetDefault("insecure", true)
	viper.SetDefault("api_endpoint", "https://10.0.0.1")
	viper.SetDefault("username", "")
	viper.SetDefault("password", "")
	viper.SetDefault("api_key", "")
	viper.SetDefault("device_mac_address", "")

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// InitFlags sets up command line flags using cobra
func InitFlags(cmd *cobra.Command) {
	// Config file flag
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/unifi-rpc.yaml)")
	
	// Server flags
	cmd.PersistentFlags().Int("port", 5000, "port to listen on")
	cmd.PersistentFlags().String("address", "0.0.0.0", "address to listen on")
	
	// Authentication flags  
	cmd.PersistentFlags().String("username", "", "UniFi controller username")
	cmd.PersistentFlags().String("password", "", "UniFi controller password")
	cmd.PersistentFlags().String("api-key", "", "UniFi controller API key (preferred over username/password)")
	
	// Connection flags
	cmd.PersistentFlags().String("api-endpoint", "https://10.0.0.1", "UniFi controller API endpoint")
	cmd.PersistentFlags().Bool("insecure", true, "allow insecure TLS connections")

	// Device flags
	cmd.PersistentFlags().String("device-mac-address", "", "Global device MAC address for the network switch (can be overridden by X-MAC-Address header)")

	// Bind flags to viper
	viper.BindPFlag("port", cmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("address", cmd.PersistentFlags().Lookup("address"))
	viper.BindPFlag("username", cmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", cmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("api_key", cmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("api_endpoint", cmd.PersistentFlags().Lookup("api-endpoint"))
	viper.BindPFlag("insecure", cmd.PersistentFlags().Lookup("insecure"))
	viper.BindPFlag("device_mac_address", cmd.PersistentFlags().Lookup("device-mac-address"))
}

// LoadConfig loads configuration from viper
func LoadConfig() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	cfg = &config
	return &config, nil
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	return cfg
}

// validateConfig ensures the configuration is valid
func validateConfig(config *Config) error {
	// Validate that we have some form of authentication
	if config.APIKey == "" && (config.Username == "" || config.Password == "") {
		return fmt.Errorf("authentication required: provide either API_KEY or both USERNAME and PASSWORD")
	}

	// Validate API endpoint (check for empty string explicitly set, not just default)
	if config.APIEndpoint == "" {
		return fmt.Errorf("API_ENDPOINT is required")
	}

	// Validate port range
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", config.Port)
	}

	return nil
}
