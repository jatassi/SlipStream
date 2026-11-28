package drivers

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// displayClass is the device class of display adapters, whose numbered
// subkeys hold each adapter's driver values.
const displayClass = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`

// Local reads each display adapter's driver version. A subkey this user
// can't open, or one without a driver version, isn't an adapter.
func Local(string, []string) Reader {
	return func(context.Context) ([]Adapter, error) {
		class, err := registry.OpenKey(registry.LOCAL_MACHINE, displayClass, registry.ENUMERATE_SUB_KEYS)
		if errors.Is(err, registry.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("open the display adapters' key: %w", err)
		}
		defer class.Close() //nolint:errcheck // a read-only key
		names, err := class.ReadSubKeyNames(-1)
		if err != nil {
			return nil, fmt.Errorf("list the display adapters: %w", err)
		}
		var out []Adapter
		for _, name := range names {
			k, err := registry.OpenKey(class, name, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			version, _, err := k.GetStringValue("DriverVersion")
			desc, _, _ := k.GetStringValue("DriverDesc")
			vendor, _, _ := k.GetStringValue("ProviderName")
			_ = k.Close()
			if err == nil {
				out = append(out, Adapter{Name: desc, Vendor: vendor, Driver: version})
			}
		}
		return out, nil
	}
}
