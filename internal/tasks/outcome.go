package tasks

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/setting"
)

type peer struct {
	name, version string
}

type result struct {
	outcome     Outcome
	cause       string
	node, short string
	dueOnReturn bool
}

func classify(ctx context.Context, err error, place Place, p peer) result {
	switch {
	case err == nil:
		return result{outcome: Succeeded}
	case canceled(ctx):
		return result{outcome: Canceled}
	}
	if why, ok := rpc.NotSent(err); ok {
		return unsent(why, rpc.Unheld(err), place, p)
	}
	if rpc.Lost(err) {
		return result{outcome: Skipped, cause: "lost " + p.name + " mid-run", node: p.name, short: p.name + " offline", dueOnReturn: place.kind == tiedTo}
	}
	return result{outcome: Failed, cause: err.Error()}
}

func canceled(runCtx context.Context) bool {
	return runCtx.Err() != nil
}

func unsent(why rpc.Why, unheld role.Name, place Place, p peer) result {
	r := result{outcome: Skipped, node: p.name, dueOnReturn: place.kind == tiedTo}
	switch why {
	case rpc.Disconnected:
		r.cause, r.short = p.name+" is offline", p.name+" offline"
	case rpc.Mismatched:
		version := p.version
		if version == "" {
			version = "another version"
		}
		r.cause = p.name + " runs " + version
		r.short = r.cause
	case rpc.NotHolder:
		r.dueOnReturn = false
		if unheld == "" {
			r.cause = p.name + " is no longer paired"
		} else {
			r.cause = p.name + " no longer holds " + unheld.Label()
		}
		r.short = r.cause
	}
	return r
}

func rollUp(noun setting.Noun, results []result, canceled bool) (run Result, summary, cause string) {
	total := len(results)
	counts := map[Outcome]int{}
	var nodes, shorts []string
	for _, r := range results {
		counts[r.outcome]++
		if r.outcome == Failed && cause == "" {
			cause = r.cause
		}
		if r.outcome == Skipped {
			nodes = appendNew(nodes, r.node)
			shorts = appendNew(shorts, r.short)
		}
	}
	nouns := noun.Of(int64(total))
	done := fmt.Sprintf("%d of %d %s", counts[Succeeded], total, nouns)
	switch {
	case canceled:
		return RunCanceled, done, ""
	case counts[Succeeded] == total:
		return RunSucceeded, fmt.Sprintf("%d %s", total, nouns), ""
	case counts[Skipped] == total:
		return RunSkipped, "Waiting for " + strings.Join(nodes, " and "), ""
	case counts[Succeeded] == 0 && counts[Failed] > 0:
		return RunFailed, fmt.Sprintf("%d of %d %s failed", counts[Failed], total, nouns), cause
	}
	parts := []string{done}
	if counts[Failed] > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", counts[Failed]))
	}
	return RunPartial, strings.Join(append(parts, shorts...), ", "), cause
}

func appendNew(list []string, s string) []string {
	if slices.Contains(list, s) {
		return list
	}
	return append(list, s)
}
