package health

import (
	"context"
	"slices"
)

// Join is one category's Source made of several: their items and unsure
// keys together, the earliest Next, and each Check that is set.
func Join(sources ...Source) Source {
	var checks []func(context.Context, Key) ([]Unreached, error)
	for _, s := range sources {
		if s.Check != nil {
			checks = append(checks, s.Check)
		}
	}
	joined := Source{Answer: func(ctx context.Context) (Answer, error) { return joinAnswers(ctx, sources) }}
	if len(checks) > 0 {
		joined.Check = func(ctx context.Context, item Key) ([]Unreached, error) { return joinChecks(ctx, checks, item) }
	}
	return joined
}

func joinAnswers(ctx context.Context, sources []Source) (Answer, error) {
	var out Answer
	for _, s := range sources {
		a, err := s.Answer(ctx)
		if err != nil {
			return Answer{}, err
		}
		out.Items = append(out.Items, a.Items...)
		out.Unsure = append(out.Unsure, a.Unsure...)
		if !a.Next.IsZero() && (out.Next.IsZero() || a.Next.Before(out.Next)) {
			out.Next = a.Next
		}
	}
	return out, nil
}

func joinChecks(ctx context.Context, checks []func(context.Context, Key) ([]Unreached, error), item Key) ([]Unreached, error) {
	var out []Unreached
	for _, check := range checks {
		u, err := check(ctx, item)
		if err != nil {
			return nil, err
		}
		out = append(out, u...)
	}
	return slices.Clip(out), nil
}
