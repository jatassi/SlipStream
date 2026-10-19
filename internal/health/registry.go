package health

import (
	"errors"
	"fmt"
	"slices"
)

// registered panics unless all is IDs, in order, each declared in full: a
// registry mistake is a build mistake.
func registered(all []Category) {
	ids := make([]ID, len(all))
	for i, c := range all {
		ids[i] = c.ID
		if c.Label == "" || c.Phrase == "" || c.Answer == nil {
			panic(fmt.Sprintf("health: category %q needs a Label, a Phrase and an Answer", c.ID))
		}
	}
	if !slices.Equal(ids, IDs()) {
		panic(fmt.Sprintf("health: the categories are %q but ids.go declares %q, in that order", ids, IDs()))
	}
}

// valid refuses an answer that breaks the source contract. The evaluator
// counts it as a failed answer, so it never clears anything.
func valid(c Category, a Answer) error {
	seen := map[Key]bool{}
	key := func(k Key) error {
		if k == "" {
			return errors.New("an item has no key")
		}
		if seen[k] {
			return fmt.Errorf("key %s is answered twice", k)
		}
		seen[k] = true
		return nil
	}
	says := func(k Key, s Says) error {
		if s.Subject == "" || s.Text == "" {
			return fmt.Errorf("item %s needs a subject and a text", k)
		}
		for _, f := range s.Detail {
			if f.Label == "" || f.set() != 1 {
				return fmt.Errorf("item %s: fact %q needs a label and exactly one of text, bytes and time", k, f.Label)
			}
		}
		switch a := s.Action; {
		case a.kind == ActionCheck && c.Check == nil:
			return fmt.Errorf("item %s offers Check now, but %s has no Check", k, c.ID)
		case a.kind == ActionRun && a.task == "", a.kind == ActionLink && (a.href == "" || a.label == ""):
			return fmt.Errorf("item %s: its action is incomplete", k)
		}
		return nil
	}
	for _, it := range a.Items {
		if err := key(it.Key); err != nil {
			return err
		}
		if it.Level != Warn && it.Level != Fail {
			return fmt.Errorf("item %s has level %d, not warn or fail", it.Key, it.Level)
		}
		if err := says(it.Key, it.Says); err != nil {
			return err
		}
	}
	for _, u := range a.Unsure {
		if err := key(u.Key); err != nil {
			return err
		}
		if err := says(u.Key, u.Says); err != nil {
			return err
		}
	}
	return nil
}
