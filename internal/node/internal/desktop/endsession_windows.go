package desktop

import (
	"errors"
	"fmt"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32                     = windows.NewLazySystemDLL("user32.dll")
	getWindowLongPtr           = user32.NewProc("GetWindowLongPtrW")
	setWindowLongPtr           = user32.NewProc("SetWindowLongPtrW")
	callWindowProc             = user32.NewProc("CallWindowProcW")
	shutdownBlockReasonCreate  = user32.NewProc("ShutdownBlockReasonCreate")
	shutdownBlockReasonDestroy = user32.NewProc("ShutdownBlockReasonDestroy")
)

const (
	wmQueryEndSession = 0x0011
	wmEndSession      = 0x0016
	// gwlpWndProc is GWLP_WNDPROC, -4.
	gwlpWndProc = ^uintptr(3)
	// systrayClass is the window class fyne.io/systray registers for its
	// hidden window.
	systrayClass = "SystrayClass"
)

// watchSessionEnd subclasses systray's hidden window, so a sign-out, a
// shutdown or the Restart Manager runs the graceful stop and holds the
// session until it finishes or its deadline passes. Once user32 is loaded
// the console control handler never sees those events, so the window is the
// only place they arrive. systray's own handler would stop the app on any
// WM_ENDSESSION, even one saying another app canceled the shutdown, so
// neither message reaches it.
func watchSessionEnd(app App) error {
	hwnd, err := systrayWindow()
	if err != nil {
		return err
	}
	var prev atomic.Uintptr
	proc := windows.NewCallback(func(h windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
		switch msg {
		case wmQueryEndSession:
			reason, _ := windows.UTF16PtrFromString("Stopping SlipStream")
			_, _, _ = shutdownBlockReasonCreate.Call(uintptr(h), uintptr(unsafe.Pointer(reason))) //nolint:gosec // G103: a UTF-16 string for a Win32 call
			return 1
		case wmEndSession:
			if wParam != 0 {
				app.Stop()
				<-app.Done()
			}
			_, _, _ = shutdownBlockReasonDestroy.Call(uintptr(h))
			return 0
		}
		r, _, _ := callWindowProc.Call(prev.Load(), uintptr(h), uintptr(msg), wParam, lParam)
		return r
	})
	old, _, err := getWindowLongPtr.Call(uintptr(hwnd), gwlpWndProc)
	if old == 0 {
		return fmt.Errorf("read the tray window's procedure: %w", err)
	}
	prev.Store(old)
	if r, _, err := setWindowLongPtr.Call(uintptr(hwnd), gwlpWndProc, proc); r == 0 {
		return fmt.Errorf("subclass the tray window: %w", err)
	}
	return nil
}

// systrayWindow is this process's top-level window of systray's class.
func systrayWindow() (windows.HWND, error) {
	pid := windows.GetCurrentProcessId()
	var found windows.HWND
	each := windows.NewCallback(func(h windows.HWND, _ uintptr) uintptr {
		var owner uint32
		if _, err := windows.GetWindowThreadProcessId(h, &owner); err != nil || owner != pid {
			return 1
		}
		var name [len(systrayClass) + 1]uint16
		n, err := windows.GetClassName(h, &name[0], int32(len(name)))
		if err != nil || windows.UTF16ToString(name[:n]) != systrayClass {
			return 1
		}
		found = h
		return 0
	})
	// EnumWindows reports an error whenever the callback stops it early.
	_ = windows.EnumWindows(each, nil)
	if found == 0 {
		return 0, errors.New("no " + systrayClass + " window in this process")
	}
	return found, nil
}
