//go:build darwin

package firewall

import (
	"context"
	"os/exec"
	"strings"
)

// checkFirewall checks macOS firewall status for the specified port.
// macOS has two firewalls: Application Firewall (socketfilterfw) and pf (packet filter).
// We check the Application Firewall as it's the user-facing one.
// Returns: firewallEnabled, portAllowed, firewallName, error
func checkFirewall(ctx context.Context, port int) (bool, bool, string, error) {
	firewallName := "macOS Firewall"

	// Check Application Firewall status using socketfilterfw
	enabled, err := isMacOSFirewallEnabled(ctx)
	if err != nil {
		// If we can't check, assume firewall is not blocking
		return false, true, firewallName, err
	}

	if !enabled {
		// Firewall disabled means all ports are accessible
		return false, true, firewallName, nil
	}

	// Check if stealth mode is enabled (blocks all incoming connections)
	stealthMode, err := isMacOSStealthModeEnabled(ctx)
	if err != nil {
		return enabled, false, firewallName, err
	}

	if stealthMode {
		// Stealth mode blocks all unsolicited connections
		return enabled, false, firewallName, nil
	}

	// Check if "Block all incoming connections" is enabled
	blockAll, err := isMacOSBlockAllEnabled(ctx)
	if err != nil {
		return enabled, false, firewallName, err
	}

	if blockAll {
		return enabled, false, firewallName, nil
	}

	// If firewall is on but not in stealth/block-all mode,
	// it allows signed apps by default. SlipStream should be allowed.
	return enabled, true, firewallName, nil
}

// isMacOSFirewallEnabled checks if macOS Application Firewall is enabled.
func isMacOSFirewallEnabled(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Output: "Firewall is enabled. (State = 1)" or "Firewall is disabled. (State = 0)"
	outputStr := strings.ToLower(string(output))
	return strings.Contains(outputStr, "enabled") || strings.Contains(outputStr, "state = 1"), nil
}

// isMacOSStealthModeEnabled checks if stealth mode is enabled.
func isMacOSStealthModeEnabled(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "/usr/libexec/ApplicationFirewall/socketfilterfw", "--getstealthmode")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Output: "Stealth mode enabled" or "Stealth mode disabled"
	outputStr := strings.ToLower(string(output))
	return strings.Contains(outputStr, "enabled"), nil
}

// isMacOSBlockAllEnabled checks if "Block all incoming connections" is enabled.
func isMacOSBlockAllEnabled(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "/usr/libexec/ApplicationFirewall/socketfilterfw", "--getblockall")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Output: "Block all ENABLED" or "Block all DISABLED"
	outputStr := strings.ToLower(string(output))
	return strings.Contains(outputStr, "enabled"), nil
}
