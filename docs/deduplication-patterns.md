# Logic Duplication: Recognition and Elimination

## The Anti-Pattern

Multiple functions contain the same decision-making logic but differ only in their outputs or side effects.

## Recognition

### Symptoms

1. **Structural similarity**: Two or more functions with the same control flow (loops, conditionals, sorting) but different final actions
2. **Parallel maintenance**: Bug fixes or feature changes require updating multiple locations
3. **Subtle behavioral drift**: Functions that "should" behave the same but don't due to independent evolution

### Code Smells

```go
// Smell: Same iteration + filtering + sorting pattern
func functionA(...) TypeA {
    for _, item := range items {
        if condition(item) {
            // evaluate...
        }
    }
    sort.Slice(results, sameComparator)
    // build TypeA
}

func functionB(...) TypeB {
    for _, item := range items {
        if condition(item) {  // SAME condition
            // evaluate...     // SAME evaluation
        }
    }
    sort.Slice(results, sameComparator)  // SAME sort
    // build TypeB (or perform side effect)
}
```

### Detection Questions

1. Do these functions make the same decisions?
2. If I fix a bug in one, must I fix it in the other?
3. Could these functions produce inconsistent results for the same input?

## The Fix Algorithm

### Step 1: Identify the Decision Boundary

Find where "deciding what to do" ends and "doing it" begins.

**Before:**
```go
func process(...) {
    // DECISION LOGIC (duplicated)
    best := findBest(items)
    if conflict(best) {
        best = findAlternative(items)
    }

    // ACTION (varies per function)
    performAction(best)
}
```

### Step 2: Define a Decision Result Type

Create a type that captures all decisions without performing any action.

```go
type Decision struct {
    SelectedItem  *Item
    AlternativeUsed bool
    Reason        string  // Why this decision was made (useful for preview/logging)
}
```

### Step 3: Extract Decision Logic

Move all decision-making into a single function that returns the decision type.

```go
func decide(items []Item) Decision {
    best := findBest(items)
    if conflict(best) {
        alt := findAlternative(items)
        return Decision{SelectedItem: alt, AlternativeUsed: true}
    }
    return Decision{SelectedItem: best}
}
```

### Step 4: Create Thin Action Functions

Each original function becomes a thin wrapper: call decide(), then act on result.

```go
func functionA(...) TypeA {
    decision := decide(items)
    return TypeA{Field: decision.SelectedItem, ...}  // Convert to output type
}

func functionB(...) {
    decision := decide(items)
    database.Save(decision.SelectedItem)  // Perform side effect
}
```

### Verification

After refactoring:
- Decision logic exists in exactly ONE place
- Action functions contain NO conditional logic about what to select
- Action functions are trivial (direct mapping or single operation)

## Example

### Before: Duplicated Logic

```go
func generatePreview(files []File, slots []Slot) []Preview {
    var previews []Preview
    sort.Slice(files, func(i, j int) bool { return files[i].Score > files[j].Score })
    taken := make(map[int64]bool)

    for _, f := range files {
        best := findBestSlot(f, slots)
        if taken[best.ID] {
            best = findAlternativeSlot(f, slots, taken)
        }
        if best != nil {
            taken[best.ID] = true
        }
        previews = append(previews, Preview{File: f, Slot: best})
    }
    return previews
}

func executeAssignment(files []File, slots []Slot) int {
    assigned := 0
    sort.Slice(files, func(i, j int) bool { return files[i].Score > files[j].Score })  // DUPLICATED
    taken := make(map[int64]bool)  // DUPLICATED

    for _, f := range files {
        best := findBestSlot(f, slots)  // DUPLICATED
        if taken[best.ID] {              // DUPLICATED
            best = findAlternativeSlot(f, slots, taken)  // DUPLICATED
        }
        if best != nil {
            taken[best.ID] = true  // DUPLICATED
            db.Assign(f.ID, best.ID)
            assigned++
        }
    }
    return assigned
}
```

### After: Single Decision Function

```go
type Assignment struct {
    FileID   int64
    SlotID   *int64
    Conflict string
}

// SINGLE source of truth for assignment decisions
func resolveAssignments(files []File, slots []Slot) []Assignment {
    sort.Slice(files, func(i, j int) bool { return files[i].Score > files[j].Score })
    taken := make(map[int64]bool)
    var assignments []Assignment

    for _, f := range files {
        best := findBestSlot(f, slots)
        if best != nil && taken[best.ID] {
            best = findAlternativeSlot(f, slots, taken)
        }

        a := Assignment{FileID: f.ID}
        if best != nil {
            a.SlotID = &best.ID
            taken[best.ID] = true
        } else {
            a.Conflict = "no available slot"
        }
        assignments = append(assignments, a)
    }
    return assignments
}

// Thin wrapper: decision -> preview format
func generatePreview(files []File, slots []Slot) []Preview {
    assignments := resolveAssignments(files, slots)
    previews := make([]Preview, len(assignments))
    for i, a := range assignments {
        previews[i] = Preview{FileID: a.FileID, SlotID: a.SlotID, Conflict: a.Conflict}
    }
    return previews
}

// Thin wrapper: decision -> side effect
func executeAssignment(files []File, slots []Slot) int {
    assignments := resolveAssignments(files, slots)
    assigned := 0
    for _, a := range assignments {
        if a.SlotID != nil {
            db.Assign(a.FileID, *a.SlotID)
            assigned++
        }
    }
    return assigned
}
```

## Common Variations

| Variation | Recognition | Solution |
|-----------|-------------|----------|
| Preview vs Execute | Same logic, different output types | Extract decision, thin converters |
| Validate vs Apply | Same checks, one returns errors, one acts | Extract validation result type |
| Estimate vs Calculate | Same algorithm, one approximate | Single algorithm, precision parameter |
| Plan vs Run | Same traversal, different actions | Extract plan as data structure |

## Anti-Patterns

1. **Partial extraction**: Extracting only some shared logic leaves remaining duplication
2. **Mode flags**: `func process(dryRun bool)` hides duplication inside conditionals
3. **Callback injection**: `func process(action func(x))` obscures what decisions are made
4. **Copy-paste with modifications**: Creates drift over time
