package main

import (
	"sort"
	"fmt"

	"github.com/pkg/errors"
)

// PlatformID represents target platform
type PlatformID string

// String retrieves the string representation of PlatformID
func (p PlatformID) String() string {
	return string(p)
}

// PlatformIDSet is a set of PlatformIDs.
type PlatformIDSet struct {
	set map[string]bool
}

// Contains returns true if `id` is in the set.
func (ps *PlatformIDSet) Contains(id interface{}) bool {
	if ps.set == nil {
		return false
	}
	var ok bool
	switch v := id.(type) {
	case string:
		_, ok = ps.set[v]
	case fmt.Stringer:
		_, ok = ps.set[v.String()]
	default:
		panic("failed to convert id into PlatformID")
	}
	return ok
}

// Add adds `id` to set.
func (ps *PlatformIDSet) Add(id PlatformID) *PlatformIDSet {
	if ps.set == nil {
		ps.set = make(map[string]bool)
	}
	ps.set[string(id)] = true
	return ps
}

// Equals checks the equality of two `PlatformIDSet`s.
func (ps *PlatformIDSet) Equals(other PlatformIDSet) bool {
	if len(ps.set) != len(other.set) {
		return false
	}
	for k := range ps.set {
		if _, ok := other.set[k]; !ok {
			return false
		}
	}
	return true
}

// ToSlice convert to slice.
func (ps *PlatformIDSet) ToSlice() []PlatformID {
	result := make([]PlatformID, 0, len(ps.set))
	for k := range ps.set {
		result = append(result, PlatformID(k))
	}
	sort.Slice(result, func(i int, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

// MarshalYAML is called while marshaling PlatformIDSet.
func (ps *PlatformIDSet) MarshalYAML() (interface{}, error) {
	if len(ps.set) == 0 {
		return []string{}, nil
	}
	result := make([]string, 0, len(ps.set))
	for k := range ps.set {
		result = append(result, string(k))
	}
	sort.Strings(result)
	return result, nil
}

// UnmarshalYAML is called while unmarshaling PlatformIDSet.
func (ps *PlatformIDSet) UnmarshalYAML(unmarshaler func(interface{}) error) error {
	var ids interface{}

	err := unmarshaler(&ids)
	if err != nil {
		errors.Wrapf(err, "failed to unmarshal PlatformIDSet")
	}
	ps.set = make(map[string]bool)
	switch v := ids.(type) {
	case string:
		ps.set[v] = true
	case []interface{}:
		for _, val := range v {
			ps.set[val.(string)] = true
		}
	case nil:
		/* NO-OP */
	default:
		return errors.Errorf("unexpected type %v found", v)
	}
	return nil
}

