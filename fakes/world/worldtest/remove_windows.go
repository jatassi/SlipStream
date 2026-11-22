package worldtest

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func heldBriefly(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_ACCESS_DENIED)
}

// withHolders is err, a hold that outlasted removeAll's wait, with the
// processes Restart Manager says still hold its file.
func withHolders(err error) error {
	var pathErr *fs.PathError
	if !heldBriefly(err) || !errors.As(err, &pathErr) {
		return err
	}
	names, herr := holders(pathErr.Path)
	if herr != nil {
		return fmt.Errorf("%w; its holders are unknown: %w", err, herr)
	}
	return fmt.Errorf("%w; held by %s", err, strings.Join(names, ", "))
}

var (
	rstrtmgr            = windows.NewLazySystemDLL("rstrtmgr.dll")
	rmStartSession      = rstrtmgr.NewProc("RmStartSession")
	rmRegisterResources = rstrtmgr.NewProc("RmRegisterResources")
	rmGetList           = rstrtmgr.NewProc("RmGetList")
	rmEndSession        = rstrtmgr.NewProc("RmEndSession")
)

// rmProcessInfo is Restart Manager's RM_PROCESS_INFO.
type rmProcessInfo struct {
	PID              uint32
	StartTime        windows.Filetime
	AppName          [256]uint16
	ServiceShortName [64]uint16
	AppType          uint32
	AppStatus        uint32
	TSSessionID      uint32
	Restartable      int32
}

// holders names each process that has path open, as "name (PID n)".
func holders(path string) ([]string, error) {
	var session uint32
	var key [33]uint16
	if r, _, _ := rmStartSession.Call(uintptr(unsafe.Pointer(&session)), 0, uintptr(unsafe.Pointer(&key[0]))); r != 0 { //nolint:gosec // G103: pointers for a Win32 call
		return nil, fmt.Errorf("RmStartSession: %w", syscall.Errno(r))
	}
	defer func() { _, _, _ = rmEndSession.Call(uintptr(session)) }()
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("RmRegisterResources: %w", err)
	}
	if r, _, _ := rmRegisterResources.Call(uintptr(session), 1, uintptr(unsafe.Pointer(&file)), 0, 0, 0, 0); r != 0 { //nolint:gosec // G103: pointers for a Win32 call
		return nil, fmt.Errorf("RmRegisterResources: %w", syscall.Errno(r))
	}
	var infos []rmProcessInfo
	for {
		var needed, reasons uint32
		n := uint32(len(infos)) //nolint:gosec // a few processes
		var first *rmProcessInfo
		if n > 0 {
			first = &infos[0]
		}
		r, _, _ := rmGetList.Call(uintptr(session), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&n)), uintptr(unsafe.Pointer(first)), uintptr(unsafe.Pointer(&reasons))) //nolint:gosec // G103: pointers for a Win32 call
		if syscall.Errno(r) == windows.ERROR_MORE_DATA {
			infos = make([]rmProcessInfo, needed)
			continue
		}
		if r != 0 {
			return nil, fmt.Errorf("RmGetList: %w", syscall.Errno(r))
		}
		names := make([]string, n)
		for i, p := range infos[:n] {
			names[i] = fmt.Sprintf("%s (PID %d)", windows.UTF16ToString(p.AppName[:]), p.PID)
		}
		return names, nil
	}
}
