package converting

import "strings"

// classes are ffmpeg's error lines by the fault they mean, in the order
// Classify tries them: a device that can't open also fails its encoder,
// and a filter chain that can't start leaves its encoder unopened.
var classes = []struct {
	code  string
	lines []string
}{
	{DiskFull, []string{"No space left on device", "There is not enough space on the disk"}},
	{Unreadable, []string{"Error opening input", "Invalid data found when processing input", "moov atom not found"}},
	{DeviceFault, []string{
		"Device creation failed", "No VA display found", "Cannot load libcuda",
		"Cannot load nvcuda.dll", "Could not dynamically load CUDA", "Error creating a MFX session", "No device available for decoder",
		"No capable devices found", "Failed to create Direct3D device",
	}},
	{Filter, []string{"Error reinitializing filters", "Error initializing filter", "Error applying option", "No such filter", "Error configuring filter"}},
	{Encoder, []string{"Error while opening encoder", "Could not open encoder", "Unknown encoder", "Encoder not found"}},
}

// Classify maps an exit to a fault code by its error output's patterns,
// crash when none matches.
func Classify(e Exit) string {
	for _, c := range classes {
		for _, line := range c.lines {
			if strings.Contains(e.Tail, line) {
				return c.code
			}
		}
	}
	return Crash
}
