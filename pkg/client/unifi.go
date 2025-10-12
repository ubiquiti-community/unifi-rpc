package client

import (
	"context"
	"encoding/csv"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type PowerState int

const (
	PowerOff PowerState = iota
	PowerOn
	PoweringOff
	PoweringOn
)

var stateName = map[PowerState]string{
	PowerOff:    "off",
	PowerOn:     "on",
	PoweringOff: "powering off",
	PoweringOn:  "powering on",
}

func (ps PowerState) String() string {
	return stateName[ps]
}

// PoEPortStatus represents the detailed PoE status for a single port.
type PoEPortStatus struct {
	Port       int     // Port number
	OpMode     string  // Operating mode (e.g., "Auto", "Off")
	HpMode     string  // High power mode (e.g., "Dot3at", "Dot3af")
	PwrLimit   int     // Power limit in milliwatts
	Class      string  // PoE class (e.g., "Class 4")
	PoEPwr     string  // PoE power status (e.g., "On", "Off")
	PwrGood    string  // Power good status (e.g., "Good", "Bad")
	PowerWatts float64 // Power consumption in watts
	VoltageV   float64 // Voltage in volts
	CurrentMA  float64 // Current in milliamps
}

// PoEStatus represents the complete PoE status output including all ports.
type PoEStatus struct {
	TotalPowerLimit int             // Total power limit in milliwatts
	Ports           []PoEPortStatus // Status for each port
}

// Config holds the configuration for connecting to a Unifi switch via SSH.
type Config struct {
	// Host is the IP address or hostname of the Unifi switch
	Host string
	// Port is the SSH port (default: 22)
	Port int
	// Username for SSH authentication
	Username string
	// PrivateKey is the SSH private key for authentication
	PrivateKey []byte
	// HostKeyCallback for host key verification (use ssh.InsecureIgnoreHostKey() for testing)
	HostKeyCallback ssh.HostKeyCallback
	// Timeout for SSH operations
	Timeout time.Duration
}

// Client provides SSH-based power management for Unifi switches.
type Client struct {
	config    *Config
	sshConfig *ssh.ClientConfig
}

// PortMapping maps MAC addresses to switch ports.
type PortMapping map[string]int // MAC address (string) -> Port ID (int)

// NewClient creates a new Unifi SSH client.
func NewClient(config *Config) (*Client, error) {
	if config.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if config.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if len(config.PrivateKey) == 0 {
		return nil, fmt.Errorf("private key is required")
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Set default values
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HostKeyCallback == nil {
		config.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: config.HostKeyCallback,
		Timeout:         config.Timeout,
	}

	return &Client{
		config:    config,
		sshConfig: sshConfig,
	}, nil
}

// executeCommand executes a command on the Unifi switch via SSH.
func (c *Client) executeCommand(ctx context.Context, command string) (string, error) {
	// Create SSH connection
	conn, err := ssh.Dial(
		"tcp",
		net.JoinHostPort(c.config.Host, strconv.Itoa(c.config.Port)),
		c.sshConfig,
	)
	if err != nil {
		return "", fmt.Errorf("failed to connect to SSH server: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer func() {
		_ = session.Close()
	}()

	// Execute the command with context timeout
	done := make(chan error, 1)
	var output []byte

	go func() {
		output, err = session.CombinedOutput(command)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("command execution cancelled: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("command execution failed: %w", err)
		}
	}

	return string(output), nil
}

// GetPortPowerState gets the current power state of a specific port.
func (c *Client) GetPortPowerState(ctx context.Context, portID int) (PowerState, error) {
	status, err := c.GetPoEStatus(ctx, portID)
	if err != nil {
		return PowerOff, fmt.Errorf("failed to get PoE status: %w", err)
	}

	for _, port := range status.Ports {
		if port.Port == portID {
			switch strings.ToLower(port.PoEPwr) {
			case "on":
				return PowerOn, nil
			case "off":
				return PowerOff, nil
			case "powering on":
				return PoweringOn, nil
			case "powering off":
				return PoweringOff, nil
			default:
				return PowerOff, fmt.Errorf("unknown power state: %s", port.PoEPwr)
			}
		}
	}

	return PowerOff, fmt.Errorf("port %d not found in status", portID)
}

// SetPortPower sets the power state of a specific port.
func (c *Client) SetPortPower(ctx context.Context, portID int, state PowerState) error {
	var command string

	switch state {
	case PowerOn, PoweringOn:
		command = fmt.Sprintf("swctrl poe set auto id %d", portID)
	case PowerOff, PoweringOff:
		command = fmt.Sprintf("swctrl poe set off id %d", portID)
	default:
		return fmt.Errorf("unsupported power state: %s", state.String())
	}

	_, err := c.executeCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to set port power state: %w", err)
	}

	return nil
}

// RestartPortPower restarts (power cycles) a specific port.
func (c *Client) RestartPortPower(ctx context.Context, portID int) error {
	command := fmt.Sprintf("swctrl poe restart id %d", portID)
	_, err := c.executeCommand(ctx, command)
	if err != nil {
		return fmt.Errorf("failed to restart port power: %w", err)
	}

	return nil
}

// GetPoEStatus gets the detailed PoE status for a specific port.
func (c *Client) GetPoEStatus(ctx context.Context, portID int) (*PoEStatus, error) {
	command := fmt.Sprintf("swctrl poe show id %d", portID)
	output, err := c.executeCommand(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("failed to get PoE status: %w", err)
	}

	return parsePoEStatus(output)
}

// parsePoEStatus parses the output from "swctrl poe show id X" command.
func parsePoEStatus(output string) (*PoEStatus, error) {
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("insufficient output lines")
	}

	status := &PoEStatus{}

	// Parse the total power limit from the first line
	// Format: "Total Power Limit(mW): -1096017060"
	for _, line := range lines {
		if strings.Contains(line, "Total Power Limit") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				limitStr := strings.TrimSpace(parts[1])
				limit, err := strconv.Atoi(limitStr)
				if err == nil {
					status.TotalPowerLimit = limit
				}
			}
			break
		}
	}

	// Find the data section (after the header lines)
	dataStartIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Look for the separator line with dashes
		if strings.HasPrefix(trimmed, "----") {
			dataStartIdx = i + 1
			break
		}
	}

	if dataStartIdx == -1 || dataStartIdx >= len(lines) {
		return nil, fmt.Errorf("could not find data section in output")
	}

	// Parse the TSV data
	for i := dataStartIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Use CSV reader with space delimiter to handle multiple spaces
		reader := csv.NewReader(strings.NewReader(line))
		reader.Comma = ' '             // Use space as delimiter
		reader.LazyQuotes = true       // Allow quotes
		reader.TrimLeadingSpace = true // Trim leading spaces
		reader.FieldsPerRecord = -1    // Variable number of fields

		records, err := reader.ReadAll()
		if err != nil || len(records) == 0 {
			continue
		}

		fields := records[0]
		// Filter out empty fields (due to multiple spaces)
		var nonEmptyFields []string
		for _, field := range fields {
			if field != "" {
				nonEmptyFields = append(nonEmptyFields, field)
			}
		}

		if len(nonEmptyFields) < 10 {
			continue // Skip malformed lines
		}

		// Parse the port status
		portStatus := PoEPortStatus{}

		// Port number
		if port, err := strconv.Atoi(nonEmptyFields[0]); err == nil {
			portStatus.Port = port
		}

		// OpMode
		portStatus.OpMode = nonEmptyFields[1]

		// HpMode
		portStatus.HpMode = nonEmptyFields[2]

		// PwrLimit
		if limit, err := strconv.Atoi(nonEmptyFields[3]); err == nil {
			portStatus.PwrLimit = limit
		}

		// Class (may be two fields like "Class 4")
		classIdx := 4
		if strings.HasPrefix(nonEmptyFields[classIdx], "Class") {
			if classIdx+1 < len(nonEmptyFields) {
				portStatus.Class = nonEmptyFields[classIdx] + " " + nonEmptyFields[classIdx+1]
				classIdx++ // Adjust for the extra field
			} else {
				portStatus.Class = nonEmptyFields[classIdx]
			}
		} else {
			portStatus.Class = nonEmptyFields[classIdx]
		}

		// PoEPwr
		portStatus.PoEPwr = nonEmptyFields[classIdx+1]

		// PwrGood
		portStatus.PwrGood = nonEmptyFields[classIdx+2]

		// Power (watts)
		if power, err := strconv.ParseFloat(nonEmptyFields[classIdx+3], 64); err == nil {
			portStatus.PowerWatts = power
		}

		// Voltage (volts)
		if voltage, err := strconv.ParseFloat(nonEmptyFields[classIdx+4], 64); err == nil {
			portStatus.VoltageV = voltage
		}

		// Current (milliamps)
		if current, err := strconv.ParseFloat(nonEmptyFields[classIdx+5], 64); err == nil {
			portStatus.CurrentMA = current
		}

		status.Ports = append(status.Ports, portStatus)
	}

	if len(status.Ports) == 0 {
		return nil, fmt.Errorf("no port data found in output")
	}

	return status, nil
}
