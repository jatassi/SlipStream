package firewall

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Status represents the firewall status for a port.
type Status struct {
	Port            int    `json:"port"`
	IsListening     bool   `json:"isListening"`
	FirewallAllows  bool   `json:"firewallAllows"`
	FirewallName    string `json:"firewallName,omitempty"`
	FirewallEnabled bool   `json:"firewallEnabled"`
	Message         string `json:"message,omitempty"`
	CheckedAt       string `json:"checkedAt"`
}

// Checker provides firewall status checking functionality.
type Checker struct{}

// NewChecker creates a new firewall checker.
func NewChecker() *Checker {
	return &Checker{}
}

// CheckPort checks if a port is open in the firewall for external access.
func (c *Checker) CheckPort(ctx context.Context, port int) (*Status, error) {
	status := &Status{
		Port:      port,
		CheckedAt: time.Now().Format(time.RFC3339),
	}

	// Check if the port is listening locally
	status.IsListening = c.isPortListening(port)

	// Check firewall status (OS-specific implementation)
	firewallEnabled, firewallAllows, firewallName, err := checkFirewall(ctx, port)
	if err != nil {
		status.Message = fmt.Sprintf("Could not check firewall: %v", err)
	}

	status.FirewallEnabled = firewallEnabled
	status.FirewallAllows = firewallAllows
	status.FirewallName = firewallName

	// Generate message based on status
	if !status.IsListening {
		status.Message = fmt.Sprintf("Port %d is not listening", port)
	} else if !status.FirewallEnabled {
		status.Message = "Firewall is disabled or not detected"
	} else if status.FirewallAllows {
		status.Message = fmt.Sprintf("Port %d is open in %s", port, firewallName)
	} else {
		status.Message = fmt.Sprintf("Port %d may be blocked by %s", port, firewallName)
	}

	return status, nil
}

// isPortListening checks if a port is currently listening.
func (c *Checker) isPortListening(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
