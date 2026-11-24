package host

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows/registry"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// sFalse is S_FALSE: COM was already initialized on this thread.
const sFalse = 1

// Firewall reads Windows Defender Firewall through INetFwPolicy2, which
// answers a user without elevation: the profile of the egress interface,
// or every active profile when egress is empty or has none, each one's
// switches, and its inbound rules. When COM can't be read every answer
// stays Unknown.
func Firewall(_ context.Context, egress string, asks []netmodel.Rule) ([]netmodel.Rule, error) {
	socks, err := tcpListeners()
	if err != nil {
		return nil, err
	}
	profiles, rules, err := readFirewall(egress)
	out := make([]netmodel.Rule, len(asks))
	for i, a := range asks {
		out[i] = a
		out[i].Tool = netmodel.WindowsDefender
		if err == nil {
			out[i] = fwDecide(profiles, rules, fwAsk{Port: a.Port, Family: a.Family, Program: ownerOf(socks, a.Port, a.Family)})
		}
	}
	return out, nil
}

func readFirewall(egress string) ([]fwProfileState, []fwRule, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		var oe *ole.OleError
		if !errors.As(err, &oe) || oe.Code() != sFalse {
			return nil, nil, fmt.Errorf("start COM: %w", err)
		}
	}
	defer ole.CoUninitialize()

	policy, err := dispatch("HNetCfg.FwPolicy2")
	if err != nil {
		return nil, nil, err
	}
	defer policy.Release()
	active := fwProfile(0)
	if egress != "" {
		active = categoryOf(egress)
	}
	if active == 0 {
		v, err := oleutil.GetProperty(policy, "CurrentProfileTypes")
		if err != nil {
			return nil, nil, fmt.Errorf("read the active firewall profiles: %w", err)
		}
		active = fwProfile(v.Val) //nolint:gosec // G115: a NET_FW_PROFILE_TYPE2 mask
	}
	var profiles []fwProfileState
	for _, p := range []fwProfile{fwDomain, fwPrivate, fwPublic} {
		if active&p == 0 {
			continue
		}
		on, err1 := boolProperty(policy, "FirewallEnabled", int32(p))
		blockAll, err2 := boolProperty(policy, "BlockAllInboundTraffic", int32(p))
		inbound, err3 := oleutil.GetProperty(policy, "DefaultInboundAction", int32(p))
		if err := errors.Join(err1, err2, err3); err != nil {
			return nil, nil, fmt.Errorf("read the %s firewall profile: %w", p, err)
		}
		profiles = append(profiles, fwProfileState{Profile: p, On: on, BlockAll: blockAll, DefaultAllow: inbound.Val == 1})
	}
	rules, err := inboundRules(policy)
	return profiles, rules, err
}

// netFwRuleDirIn and netFwActionAllow are NET_FW_RULE_DIR_IN and
// NET_FW_ACTION_ALLOW.
const (
	netFwRuleDirIn   = 1
	netFwActionAllow = 1
)

func inboundRules(policy *ole.IDispatch) ([]fwRule, error) {
	v, err := oleutil.GetProperty(policy, "Rules")
	if err != nil {
		return nil, fmt.Errorf("read the firewall rules: %w", err)
	}
	all := v.ToIDispatch()
	defer all.Release()
	var out []fwRule
	var unread error
	err = oleutil.ForEach(all, func(item *ole.VARIANT) error {
		r := item.ToIDispatch()
		defer r.Release()
		get := func(name string) (any, int32) {
			v, err := oleutil.GetProperty(r, name)
			if err != nil {
				unread = errors.Join(unread, fmt.Errorf("read %s: %w", name, err))
				return nil, 0
			}
			defer v.Clear()                //nolint:errcheck // frees the value's string
			return v.Value(), int32(v.Val) //nolint:gosec // G115: a LONG or enum
		}
		str := func(name string) string {
			v, _ := get(name)
			s, _ := v.(string)
			return s
		}
		num := func(name string) int32 {
			_, n := get(name)
			return n
		}
		if num("Direction") != netFwRuleDirIn {
			return nil
		}
		program, _ := registry.ExpandString(str("ApplicationName"))
		out = append(out, fwRule{
			Name:            str("Name"),
			Enabled:         num("Enabled") != 0,
			Inbound:         true,
			Allow:           num("Action") == netFwActionAllow,
			Profiles:        fwProfile(num("Profiles")),
			Protocol:        num("Protocol"),
			LocalPorts:      str("LocalPorts"),
			RemoteAddresses: str("RemoteAddresses"),
			Program:         program,
			Service:         str("ServiceName"),
			Packaged:        str("LocalAppPackageId") != "" || str("LocalUserOwner") != "",
		})
		return nil
	})
	if err = errors.Join(err, unread); err != nil {
		return nil, fmt.Errorf("read the firewall rules: %w", err)
	}
	return out, nil
}

// categoryOf is the firewall profile of the interface named iface, from
// its network connection profile; 0 when it has none.
func categoryOf(iface string) fwProfile {
	locator, err := dispatch("WbemScripting.SWbemLocator")
	if err != nil {
		return 0
	}
	defer locator.Release()
	v, err := oleutil.CallMethod(locator, "ConnectServer", ".", `root\StandardCimv2`)
	if err != nil {
		return 0
	}
	service := v.ToIDispatch()
	defer service.Release()
	v, err = oleutil.CallMethod(service, "ExecQuery", "SELECT InterfaceAlias, NetworkCategory FROM MSFT_NetConnectionProfile")
	if err != nil {
		return 0
	}
	found := v.ToIDispatch()
	defer found.Release()
	// NetworkCategory is Public, Private, then DomainAuthenticated.
	byCategory := []fwProfile{fwPublic, fwPrivate, fwDomain}
	var out fwProfile
	_ = oleutil.ForEach(found, func(item *ole.VARIANT) error {
		p := item.ToIDispatch()
		defer p.Release()
		alias, err1 := oleutil.GetProperty(p, "InterfaceAlias")
		category, err2 := oleutil.GetProperty(p, "NetworkCategory")
		if err1 == nil && err2 == nil && alias.ToString() == iface && int(category.Val) < len(byCategory) {
			out = byCategory[category.Val]
		}
		return nil
	})
	return out
}

func dispatch(progID string) (*ole.IDispatch, error) {
	unknown, err := oleutil.CreateObject(progID)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", progID, err)
	}
	defer unknown.Release()
	d, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", progID, err)
	}
	return d, nil
}

func boolProperty(d *ole.IDispatch, name string, args ...any) (bool, error) {
	v, err := oleutil.GetProperty(d, name, args...)
	if err != nil {
		return false, err //nolint:wrapcheck // the caller names the profile
	}
	return v.Val != 0, nil
}
