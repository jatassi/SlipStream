package fault

import (
	"errors"
	"fmt"
	"mime"
	"strconv"
	"strings"
	"time"
)

// Parse reads a fault in the one grammar:
//
//	refuse | reset | stall | latency=200ms | throttle=256KiB/s | cut=64KiB | tls=untrusted|expired|wrong-name
//	status=<code>[,retry-after=<dur>][,reset-header=<name>] | hang | malformed | truncate | slow-body=<rate>
//	content-type=<media type>
//	full | unmounted | hung | readonly | volume=<total>,free=<free>
func Parse(s string) (Fault, error) {
	name, value, hasValue := strings.Cut(s, "=")
	if f, ok := bare[name]; ok {
		if hasValue {
			return nil, fmt.Errorf("fault %s takes no value", name)
		}
		return f, nil
	}
	parse, ok := valued[name]
	if !ok {
		return nil, fmt.Errorf("unknown fault %q", s)
	}
	if !hasValue {
		return nil, fmt.Errorf("fault %s needs a value: %s=…", name, name)
	}
	f, err := parse(value)
	if err != nil {
		return nil, fmt.Errorf("fault %s: %w", s, err)
	}
	return f, nil
}

var bare = map[string]Fault{
	"refuse": Refuse{}, "reset": Reset{}, "stall": Stall{},
	"hang": Hang{}, "malformed": Malformed{}, "truncate": Truncate{},
	"full": Full{}, "unmounted": Unmounted{}, "hung": Hung{}, "readonly": ReadOnly{},
}

var valued = map[string]func(string) (Fault, error){
	"latency": func(v string) (Fault, error) {
		d, err := duration(v)
		return Latency(d), err
	},
	"throttle": func(v string) (Fault, error) {
		r, err := rate(v)
		return Throttle(r), err
	},
	"cut": func(v string) (Fault, error) {
		n, err := byteCount(v)
		return Cut(n), err
	},
	"tls": func(v string) (Fault, error) {
		for f, name := range tlsFailures {
			if name == v {
				return BadTLS(f), nil
			}
		}
		return nil, errors.New("want untrusted, expired or wrong-name")
	},
	"status": status,
	"volume": volume,
	"slow-body": func(v string) (Fault, error) {
		r, err := rate(v)
		return SlowBody(r), err
	},
	"content-type": func(v string) (Fault, error) {
		if t, _, err := mime.ParseMediaType(v); err != nil || !strings.Contains(t, "/") {
			return nil, fmt.Errorf("%q is not a media type such as text/html", v)
		}
		return ContentType(v), nil
	},
}

func status(v string) (Fault, error) {
	parts := strings.Split(v, ",")
	code, err := strconv.Atoi(parts[0])
	if err != nil || code < 100 || code > 599 {
		return nil, fmt.Errorf("status %q is not an HTTP status code", parts[0])
	}
	f := Status{Code: code}
	seen := map[string]bool{}
	for _, p := range parts[1:] {
		k, v, _ := strings.Cut(p, "=")
		if seen[k] {
			return nil, fmt.Errorf("%s given twice", k)
		}
		seen[k] = true
		switch k {
		case "retry-after":
			if f.RetryAfter, err = duration(v); err != nil {
				return nil, err
			}
		case "reset-header":
			if v == "" {
				return nil, errors.New("reset-header needs a header name")
			}
			f.ResetHeader = v
		default:
			return nil, fmt.Errorf("unknown status parameter %q", p)
		}
	}
	if f.ResetHeader != "" && f.RetryAfter == 0 {
		return nil, errors.New("reset-header needs retry-after")
	}
	return f, nil
}

func duration(v string) (time.Duration, error) {
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("duration %q is not a Go duration such as 200ms", v)
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration %s is not positive", v)
	}
	return d, nil
}

func rate(v string) (Rate, error) {
	n, ok := strings.CutSuffix(v, "/s")
	if !ok {
		return 0, fmt.Errorf("rate %q needs /s", v)
	}
	b, err := byteCount(n)
	if err != nil {
		return 0, err
	}
	if b == 0 {
		return 0, fmt.Errorf("rate %s is not positive", v)
	}
	return Rate(b), nil
}

var units = []struct {
	name  string
	scale int64
}{{"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10}, {"B", 1}}

func byteCount(v string) (int64, error) {
	for _, u := range units {
		if n, ok := strings.CutSuffix(v, u.name); ok {
			i, err := strconv.ParseInt(n, 10, 64)
			if err != nil || i < 0 {
				return 0, fmt.Errorf("size %q is not a count of %s", v, u.name)
			}
			return i * u.scale, nil
		}
	}
	return 0, fmt.Errorf("size %q needs a unit: B, KiB, MiB or GiB", v)
}

// size writes n in the largest unit that divides it.
func size(n int64) string {
	for _, u := range units {
		if n != 0 && n%u.scale == 0 {
			return strconv.FormatInt(n/u.scale, 10) + u.name
		}
	}
	return strconv.FormatInt(n, 10) + "B"
}
