//go:build linux

package firewall

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// checkFirewall checks Linux firewall status for the specified port.
// Supports: ufw, firewalld, and iptables (in order of preference).
// Returns: firewallEnabled, portAllowed, firewallName, error
func checkFirewall(ctx context.Context, port int) (bool, bool, string, error) {
	// Try ufw first (Ubuntu/Debian)
	if enabled, allowed, err := checkUFW(ctx, port); err == nil {
		return enabled, allowed, "ufw", nil
	}

	// Try firewalld (RHEL/CentOS/Fedora)
	if enabled, allowed, err := checkFirewalld(ctx, port); err == nil {
		return enabled, allowed, "firewalld", nil
	}

	// Fall back to iptables
	if enabled, allowed, err := checkIPTables(ctx, port); err == nil {
		return enabled, allowed, "iptables", nil
	}

	// No firewall detected or unable to check
	return false, true, "none detected", nil
}

// checkUFW checks ufw (Uncomplicated Firewall) status.
func checkUFW(ctx context.Context, port int) (bool, bool, error) {
	// Check if ufw is available and get status
	cmd := exec.CommandContext(ctx, "ufw", "status")
	output, err := cmd.Output()
	if err != nil {
		return false, false, err
	}

	outputStr := string(output)

	// Check if ufw is active
	if strings.Contains(strings.ToLower(outputStr), "status: inactive") {
		return false, true, nil
	}

	if !strings.Contains(strings.ToLower(outputStr), "status: active") {
		return false, true, nil
	}

	// UFW is active, check if port is allowed
	portStr := strconv.Itoa(port)

	// Look for rules that allow our port
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Rules look like: "8080/tcp                   ALLOW       Anywhere"
		if strings.Contains(line, portStr) && strings.Contains(strings.ToUpper(line), "ALLOW") {
			return true, true, nil
		}
	}

	// Port not explicitly allowed
	return true, false, nil
}

// checkFirewalld checks firewalld status.
func checkFirewalld(ctx context.Context, port int) (bool, bool, error) {
	// Check if firewalld is running
	cmd := exec.CommandContext(ctx, "firewall-cmd", "--state")
	output, err := cmd.Output()
	if err != nil {
		return false, false, err
	}

	if !strings.Contains(strings.ToLower(string(output)), "running") {
		return false, true, nil
	}

	// firewalld is running, check if port is open
	portStr := strconv.Itoa(port)
	cmd = exec.CommandContext(ctx, "firewall-cmd", "--query-port="+portStr+"/tcp")
	err = cmd.Run()

	// Exit code 0 means port is open
	return true, err == nil, nil
}

// checkIPTables checks iptables rules.
func checkIPTables(ctx context.Context, port int) (bool, bool, error) {
	// List INPUT chain rules
	cmd := exec.CommandContext(ctx, "iptables", "-L", "INPUT", "-n")
	output, err := cmd.Output()
	if err != nil {
		// Need root to check iptables, assume no blocking
		return false, true, err
	}

	outputStr := string(output)
	portStr := strconv.Itoa(port)

	// Check if there are any rules at all (besides default policy)
	lines := strings.Split(outputStr, "\n")
	hasRules := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Chain") || strings.HasPrefix(line, "target") {
			continue
		}
		hasRules = true

		// Check for ACCEPT rule for our port
		if strings.Contains(line, "ACCEPT") && strings.Contains(line, "dpt:"+portStr) {
			return true, true, nil
		}
	}

	if !hasRules {
		// No firewall rules, assume open
		return false, true, nil
	}

	// Has rules but no explicit allow for our port
	// Check default policy
	if strings.Contains(outputStr, "policy ACCEPT") {
		return true, true, nil
	}

	return true, false, nil
}
