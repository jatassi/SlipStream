package module

import (
	"errors"
	"fmt"
)

// NodeLevel defines a single tier in a module's hierarchy.
type NodeLevel struct {
	Name                     string
	PluralName               string
	IsRoot                   bool
	IsLeaf                   bool
	HasMonitored             bool
	Searchable               bool
	IsSpecial                bool
	SupportsMultiEntityFiles bool
	FormatVariants           []string
	Properties               map[string]string
}

// NodeSchema defines a module's complete hierarchy.
type NodeSchema struct {
	Levels []NodeLevel
}

// Validate checks schema invariants: exactly one root, exactly one leaf, ordered.
func (s NodeSchema) Validate() error {
	if len(s.Levels) == 0 {
		return errors.New("schema must have at least one level")
	}

	var rootCount, leafCount int
	names := make(map[string]bool)
	pluralNames := make(map[string]bool)

	for i, level := range s.Levels {
		rc, lc, err := validateLevel(level, i, len(s.Levels), names, pluralNames)
		if err != nil {
			return err
		}
		rootCount += rc
		leafCount += lc
	}

	if rootCount != 1 {
		return fmt.Errorf("schema must have exactly one root level, found %d", rootCount)
	}
	if leafCount != 1 {
		return fmt.Errorf("schema must have exactly one leaf level, found %d", leafCount)
	}

	if len(s.Levels) == 1 && (!s.Levels[0].IsRoot || !s.Levels[0].IsLeaf) {
		return errors.New("single-level schema must have its level marked as both root and leaf")
	}

	return nil
}

func validateLevel(level NodeLevel, idx, total int, names, pluralNames map[string]bool) (rootInc, leafInc int, err error) {
	if level.IsRoot {
		rootInc = 1
		if idx != 0 {
			return 0, 0, fmt.Errorf("root level %q must be the first level", level.Name)
		}
	}
	if level.IsLeaf {
		leafInc = 1
		if idx != total-1 {
			return 0, 0, fmt.Errorf("leaf level %q must be the last level", level.Name)
		}
	}
	if names[level.Name] {
		return 0, 0, fmt.Errorf("duplicate level name: %q", level.Name)
	}
	names[level.Name] = true
	if pluralNames[level.PluralName] {
		return 0, 0, fmt.Errorf("duplicate level plural name: %q", level.PluralName)
	}
	pluralNames[level.PluralName] = true
	return rootInc, leafInc, nil
}

// Root returns the root-level node, or an error if none exists.
func (s NodeSchema) Root() (NodeLevel, error) {
	for _, level := range s.Levels {
		if level.IsRoot {
			return level, nil
		}
	}
	return NodeLevel{}, errors.New("no root level found")
}

// Leaf returns the leaf-level node, or an error if none exists.
func (s NodeSchema) Leaf() (NodeLevel, error) {
	for _, level := range s.Levels {
		if level.IsLeaf {
			return level, nil
		}
	}
	return NodeLevel{}, errors.New("no leaf level found")
}

// Depth returns the number of levels.
func (s NodeSchema) Depth() int { return len(s.Levels) }
