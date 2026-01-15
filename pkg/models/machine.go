package models

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Port represents a switch port number.
type Port struct {
	Number int `json:"port"`
}

// MacLookupClient interface for looking up port by MAC address.
type MacLookupClient interface {
	GetPortByMacAddress(ctx context.Context, macAddress string) (int, error)
}

// GetPort extracts port number from HTTP headers.
// If X-Port header is present, it's used directly.
// If X-Port is not present but X-Mac is, the client is used to look up the port by MAC address.
// The client parameter can be nil if X-Port is always provided.
func GetPort(r *http.Request, client MacLookupClient) (*Port, error) {
	portStr := r.Header.Get("X-Port")

	// If X-Port is provided, use it
	if portStr != "" {
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

	// If X-Port is not provided, try X-Mac
	macStr := r.Header.Get("X-Mac")
	if macStr == "" {
		return nil, fmt.Errorf("either X-Port or X-Mac header is required")
	}

	// Client is required for MAC lookup
	if client == nil {
		return nil, fmt.Errorf("cannot lookup port by MAC address: client not provided")
	}

	// Trim whitespace
	macStr = strings.TrimSpace(macStr)

	// Look up port by MAC address
	portNum, err := client.GetPortByMacAddress(r.Context(), macStr)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup port by MAC address %q: %w", macStr, err)
	}

	return &Port{Number: portNum}, nil
}
