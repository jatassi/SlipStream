package conversions

// The fault codes a Conversion ends an attempt with. The holder reports
// the run's codes, the Transfers the copies', the root-side check
// short_output, streams and refused, and placement container_audio.
const (
	CodeEncoder        Code = "encoder"
	CodeFilter         Code = "filter"
	CodeDevice         Code = "device"
	CodeCrash          Code = "crash"
	CodeStalled        Code = "stalled"
	CodeShortOutput    Code = "short_output"
	CodeStreams        Code = "streams"
	CodeCopyFailed     Code = "copy_failed"
	CodeReturnFailed   Code = "return_failed"
	CodeUnreadable     Code = "unreadable"
	CodeContainerAudio Code = "container_audio"
	CodeRefused        Code = "refused"
	CodeDiskFull       Code = "disk_full"
)

// FaultRule is what a code costs: its Class, and whether the Node loses
// its proof of the Conversion's key.
type FaultRule struct {
	Class Class
	Costs bool
}

var faults = map[Code]FaultRule{
	CodeEncoder: {RunClass, true}, CodeFilter: {RunClass, true}, CodeDevice: {RunClass, true}, CodeCrash: {RunClass, true},
	CodeStalled: {RunClass, true}, CodeShortOutput: {RunClass, true}, CodeStreams: {RunClass, true},
	CodeCopyFailed: {RunClass, false}, CodeReturnFailed: {RunClass, false},
	CodeUnreadable: {SourceClass, false}, CodeContainerAudio: {SourceClass, false}, CodeRefused: {SourceClass, false},
	CodeDiskFull: {PutBackClass, false},
}

// outcomeOf is the attempt outcome a run fault closes it with.
func outcomeOf(code Code) Outcome {
	if code == CodeStalled {
		return AttemptStalled
	}
	return AttemptFailed
}
