package tags

// Diff describes the changes needed to transform current into desired.
// Both maps are assumed to be custom-tag-only (for example from OCM GET/list
// with fetchUserTagsOnly=true).
type Diff struct {
	// Add contains keys present in desired but not in current.
	Add map[string]string
	// Update contains keys present in both maps with different values (desired values).
	Update map[string]string
	// Remove contains keys present in current but not in desired.
	Remove []string
}

// Empty reports whether the diff has no changes.
func (d Diff) Empty() bool {
	return len(d.Add) == 0 && len(d.Update) == 0 && len(d.Remove) == 0
}

// Equal reports whether a and b contain the same keys and values.
// Nil and empty maps are treated as equal.
func Equal(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok || bv != v {
			return false
		}
	}
	return true
}

// ComputeDiff returns the add/update/remove sets to change current into desired.
// Nil maps are treated as empty.
func ComputeDiff(current, desired map[string]string) Diff {
	diff := Diff{
		Add:    map[string]string{},
		Update: map[string]string{},
		Remove: []string{},
	}

	for k, dv := range desired {
		cv, ok := current[k]
		if !ok {
			diff.Add[k] = dv
			continue
		}
		if cv != dv {
			diff.Update[k] = dv
		}
	}
	for k := range current {
		if _, ok := desired[k]; !ok {
			diff.Remove = append(diff.Remove, k)
		}
	}
	return diff
}
