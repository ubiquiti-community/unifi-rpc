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

	// UniFi Switch SSH connection
	SwitchHost  string `mapstructure:"switch_host"`
	SSHPort     int    `mapstructure:"ssh_port"`
	SSHUsername string `mapstructure:"ssh_username"`
	SSHKeyPath  string `mapstructure:"ssh_key_path"`
}

var (
	cfgFile string
	cfg     *Config
)

// InitConfig initializes viper configuration.
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
	viper.SetDefault("ssh_port", 22)
	viper.SetDefault("ssh_username", "root")
	viper.SetDefault("switch_host", "")
	viper.SetDefault("ssh_key_path", "") // Empty default - must be explicitly configured

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// InitFlags sets up command line flags using cobra.
func InitFlags(cmd *cobra.Command) {
	// Config file flag
	cmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/unifi-rpc.yaml)")

	// Server flags
	cmd.PersistentFlags().Int("port", 5000, "port to listen on")
	cmd.PersistentFlags().String("address", "0.0.0.0", "address to listen on")

	// SSH connection flags
	cmd.PersistentFlags().String("switch-host", "", "UniFi switch IP address or hostname")
	cmd.PersistentFlags().Int("ssh-port", 22, "SSH port")
	cmd.PersistentFlags().String("ssh-username", "root", "SSH username")
	cmd.PersistentFlags().String("ssh-key-path", "", "Path to SSH private key file")

	// Bind flags to viper
	_ = viper.BindPFlag("port", cmd.PersistentFlags().Lookup("port"))
	_ = viper.BindPFlag("address", cmd.PersistentFlags().Lookup("address"))
	_ = viper.BindPFlag("switch_host", cmd.PersistentFlags().Lookup("switch-host"))
	_ = viper.BindPFlag("ssh_port", cmd.PersistentFlags().Lookup("ssh-port"))
	_ = viper.BindPFlag("ssh_username", cmd.PersistentFlags().Lookup("ssh-username"))
	_ = viper.BindPFlag("ssh_key_path", cmd.PersistentFlags().Lookup("ssh-key-path"))
}

// LoadConfig loads configuration from viper.
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

// GetConfig returns the current configuration.
func GetConfig() *Config {
	return cfg
}

// validateConfig ensures the configuration is valid.
func validateConfig(config *Config) error {
	// Validate SSH configuration
	if config.SwitchHost == "" {
		return fmt.Errorf("switch_host is required")
	}

	if config.SSHKeyPath == "" {
		return fmt.Errorf("ssh_key_path is required")
	}

	// Validate SSH port range
	if config.SSHPort < 1 || config.SSHPort > 65535 {
		return fmt.Errorf("ssh_port must be between 1 and 65535, got %d", config.SSHPort)
	}

	// Validate server port range
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", config.Port)
	}

	return nil
}
