package models

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Port represents a switch port number.
type Port struct {
	Number int `json:"port"`
}

// GetPort extracts port number from HTTP headers
// X-Port header is required.
func GetPort(r *http.Request) (*Port, error) {
	portStr := r.Header.Get("X-Port")
	if portStr == "" {
		return nil, fmt.Errorf("X-Port header is required")
	}

	// Trim whitespace
	portStr = strings.TrimSpace(portStr)

	portNum, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port number %q: %w", portStr, err)
	}

	if portNum < 1 {
		return nil, fmt.Errorf("invalid port number: port must be positive, got %d", portNum)
	}

	return &Port{Number: portNum}, nil
}
