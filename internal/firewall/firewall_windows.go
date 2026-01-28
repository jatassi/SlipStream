//go:build windows

package firewall

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// checkFirewall checks Windows Firewall status for the specified port.
// Returns: firewallEnabled, portAllowed, firewallName, error
func checkFirewall(ctx context.Context, port int) (bool, bool, string, error) {
	firewallName := "Windows Firewall"

	// Check if Windows Firewall is enabled
	enabled, err := isWindowsFirewallEnabled(ctx)
	if err != nil {
		return false, false, firewallName, err
	}

	if !enabled {
		return false, true, firewallName, nil
	}

	// Check if the port has an allow rule
	allowed, err := isPortAllowedInWindowsFirewall(ctx, port)
	if err != nil {
		return enabled, false, firewallName, err
	}

	return enabled, allowed, firewallName, nil
}

// isWindowsFirewallEnabled checks if Windows Firewall is enabled.
func isWindowsFirewallEnabled(ctx context.Context) (bool, error) {
	// Check firewall state using netsh
	cmd := exec.CommandContext(ctx, "netsh", "advfirewall", "show", "currentprofile", "state")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Parse output - look for "State" line with "ON"
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "state") {
			return strings.Contains(strings.ToLower(line), "on"), nil
		}
	}

	return false, nil
}

// isPortAllowedInWindowsFirewall checks if a port has an inbound allow rule.
func isPortAllowedInWindowsFirewall(ctx context.Context, port int) (bool, error) {
	portStr := strconv.Itoa(port)

	// Query inbound rules for TCP port
	cmd := exec.CommandContext(ctx, "netsh", "advfirewall", "firewall", "show", "rule",
		"name=all", "dir=in", "protocol=tcp")
	output, err := cmd.Output()
	if err != nil {
		// If the command fails, we can't determine status
		return false, err
	}

	// Parse output looking for rules that allow this port
	lines := strings.Split(string(output), "\n")
	var currentRuleEnabled bool
	var currentRuleAction string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		lineLower := strings.ToLower(line)

		// Track rule enabled state
		if strings.HasPrefix(lineLower, "enabled:") {
			currentRuleEnabled = strings.Contains(lineLower, "yes")
		}

		// Track rule action
		if strings.HasPrefix(lineLower, "action:") {
			currentRuleAction = line
		}

		// Check local port
		if strings.HasPrefix(lineLower, "localport:") {
			ports := strings.TrimPrefix(lineLower, "localport:")
			ports = strings.TrimSpace(ports)

			// Check if this rule applies to our port
			if portMatches(ports, portStr) && currentRuleEnabled {
				if strings.Contains(strings.ToLower(currentRuleAction), "allow") {
					return true, nil
				}
			}
		}

		// Reset on new rule
		if strings.HasPrefix(lineLower, "rule name:") {
			currentRuleEnabled = false
			currentRuleAction = ""
		}
	}

	return false, nil
}

// portMatches checks if a port specification includes the target port.
func portMatches(portSpec, targetPort string) bool {
	// Handle "Any" case
	if strings.ToLower(portSpec) == "any" {
		return true
	}

	// Handle comma-separated ports
	parts := strings.Split(portSpec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Handle range (e.g., "8000-9000")
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				target, err3 := strconv.Atoi(targetPort)
				if err1 == nil && err2 == nil && err3 == nil {
					if target >= start && target <= end {
						return true
					}
				}
			}
		} else if part == targetPort {
			return true
		}
	}

	return false
}
