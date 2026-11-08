package regex

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/dlclark/regexp2/syntax"
)

// CompileError names the pattern, the rune offset the engine stopped at,
// and its reason, for "doesn't compile at position 14 (unterminated group)".
// Offset is .NET's RegexParseException.Offset: the runes read before the
// engine stopped, so the 1-based position of the rune it stopped at. It is
// 0 when the reason has no position, because regexp2 reports none and the
// reason is one a cut-off pattern also gives.
type CompileError struct {
	Pattern string
	Offset  int
	Reason  string
}

func (e *CompileError) Error() string {
	if e.Offset == 0 {
		return fmt.Sprintf("the pattern %s doesn't compile (%s)", quoted(e.Pattern), e.Reason)
	}
	return fmt.Sprintf("the pattern %s doesn't compile at position %d (%s)", quoted(e.Pattern), e.Offset, e.Reason)
}

// TimeoutError is a match that ran past its limit.
type TimeoutError struct {
	Pattern string
	Limit   time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("the pattern %s ran past its %v match limit", quoted(e.Pattern), e.Limit)
}

// reason is how one regexp2 error code reads. A positional code is raised
// where the engine reads the fault, so the shortest prefix of the pattern
// that fails with the same code and arguments ends where the engine
// stopped. The others are also what a pattern cut short gives, so a
// prefix can't place them.
type reason struct {
	format     string
	positional bool
}

var reasons = map[syntax.ErrorCode]reason{
	syntax.ErrInternalError:              {"the regex engine failed", false},
	syntax.ErrUnterminatedComment:        {"unterminated (?# comment", false},
	syntax.ErrInvalidCharRange:           {"a character class range can't be read", true},
	syntax.ErrInvalidRepeatSize:          {"a {n,m} count is out of range", true},
	syntax.ErrInvalidUTF8:                {"the pattern isn't UTF-8 text", true},
	syntax.ErrCaptureGroupOutOfRange:     {"a group number is out of range", true},
	syntax.ErrUnexpectedParen:            {"a ) has no matching (", true},
	syntax.ErrMissingParen:               {"unterminated group", false},
	syntax.ErrMissingBrace:               {"a { has no matching }", false},
	syntax.ErrInvalidRepeatOp:            {"a repetition follows another repetition", true},
	syntax.ErrMissingRepeatArgument:      {"*, + or ? has nothing to repeat", true},
	syntax.ErrConditionalExpression:      {"a (?(...)) condition can't be read", true},
	syntax.ErrTooManyAlternates:          {"a (?(...)) condition has more than two branches", true},
	syntax.ErrUnrecognizedGrouping:       {"(%v isn't a known group construct", true},
	syntax.ErrInvalidGroupName:           {"a group name must start with a letter, a digit or _ and end with its closing bracket", true},
	syntax.ErrCapNumNotZero:              {"a group can't be numbered 0", true},
	syntax.ErrUndefinedBackRef:           {"group %v doesn't exist", true},
	syntax.ErrUndefinedNameRef:           {"group %v doesn't exist", true},
	syntax.ErrAlternationCantCapture:     {"a (?(...)) condition can't capture or be named", true},
	syntax.ErrAlternationCantHaveComment: {"a (?(...)) condition can't be a comment", true},
	syntax.ErrMalformedReference:         {"(?(%v) ) can't be read", true},
	syntax.ErrUndefinedReference:         {"(?(%v) ) refers to a group that doesn't exist", true},
	syntax.ErrIllegalEndEscape:           {"the pattern ends with a lone \\", false},
	syntax.ErrMalformedSlashP:            {"a \\p{...} escape can't be read", true},
	syntax.ErrIncompleteSlashP:           {"unterminated \\p{...} escape", false},
	syntax.ErrUnknownSlashP:              {"%v isn't a known Unicode category or block", true},
	syntax.ErrUnrecognizedEscape:         {"\\%v isn't a known escape", true},
	syntax.ErrMissingControl:             {"\\c has no control character after it", false},
	syntax.ErrUnrecognizedControl:        {"\\c is followed by a character that isn't a control letter", true},
	syntax.ErrTooFewHex:                  {"a hex escape has too few digits", false},
	syntax.ErrInvalidHex:                 {"a hex value is above 0x10FFFF", true},
	syntax.ErrMalformedNameRef:           {"a \\k<...> reference can't be read", false},
	syntax.ErrBadClassInCharRange:        {"a range in [] can't end at \\%v", true},
	syntax.ErrUnterminatedBracket:        {"unterminated character class", false},
	syntax.ErrSubtractionMustBeLast:      {"a class subtraction must come last in []", true},
	syntax.ErrReversedCharRange:          {"the range %c-%c is in reverse order", true},
}

func compileError(pattern string, o Options, err error) *CompileError {
	var se *syntax.Error
	errors.As(err, &se)
	r := reasons[se.Code]
	e := &CompileError{Pattern: pattern, Reason: fmt.Sprintf(r.format, se.Args...)}
	if r.positional {
		e.Offset = stoppedAt(pattern, o, se)
	}
	return e
}

func stoppedAt(pattern string, o Options, want *syntax.Error) int {
	runes := []rune(pattern)
	for n := 1; n < len(runes); n++ {
		rewritten, _ := rewriteBlocks(string(runes[:n]))
		_, err := regexp2.Compile(rewritten, o.flags())
		var se *syntax.Error
		if errors.As(err, &se) && se.Code == want.Code && reflect.DeepEqual(se.Args, want.Args) {
			return n
		}
	}
	return len(runes)
}

// quoted is text a person typed, such as a name or a pattern, in the
// straight quotes WRITING.md sets and exactly as typed. %q would escape
// each backslash in a pattern.
func quoted(s string) string { return `"` + s + `"` }
