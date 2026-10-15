package desktop

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <stdlib.h>
#include <CoreFoundation/CoreFoundation.h>

static void alert(const char *title, const char *text) {
	CFStringRef t = CFStringCreateWithCString(NULL, title, kCFStringEncodingUTF8);
	CFStringRef m = CFStringCreateWithCString(NULL, text, kCFStringEncodingUTF8);
	CFOptionFlags response;
	CFUserNotificationDisplayAlert(0, kCFUserNotificationStopAlertLevel, NULL, NULL, NULL, t, m, NULL, NULL, NULL, &response);
	if (t) CFRelease(t);
	if (m) CFRelease(m);
}
*/
import "C"

import "unsafe"

// Alert shows a modal error alert. CoreFoundation's needs no running
// NSApplication, and the app fails before its menu bar item exists.
func Alert(title, text string) {
	t, m := C.CString(title), C.CString(text)
	defer C.free(unsafe.Pointer(t))
	defer C.free(unsafe.Pointer(m))
	C.alert(t, m)
}
