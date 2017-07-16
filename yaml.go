// Schema definitions for values from `make.yml`.
package main

import (
	"fmt"
	"strings"

	"sort"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// PlatformID represents target platform
type PlatformID string

// String retrieves the string representation of PlatformID
func (p PlatformID) String() string {
	return string(p)
}

// Data format make.yml top structure
type Data struct {
	Target        []Target     `yaml:",flow"`
	Include       []StringList `yaml:",flow"`
	Variable      []Variable   `yaml:",flow"`
	Define        []StringList `yaml:",flow"`
	Option        []StringList `yaml:",flow"`
	ArchiveOption []StringList `yaml:"archive_option,flow"`
	ConvertOption []StringList `yaml:"convert_option,flow"`
	LinkOption    []StringList `yaml:"link_option,flow"`
	LinkDepend    []StringList `yaml:"link_depend,flow"`
	Libraries     []StringList `yaml:",flow"`
	Prebuild      []Build      `yaml:",flow"`
	Postbuild     []Build      `yaml:",flow"`
	Source        []StringList `yaml:",flow"`
	Headers       []StringList `yaml:"header,flow"`
	ConvertList   []StringList `yaml:"convert_list,flow"`
	Subdirs       []StringList `yaml:"subdir,flow"`
	Tests         []StringList `yaml:",flow"`
	Other         []Other      `yaml:",flow"`
	SubNinja      []StringList `yaml:",flow"`
}

// Target make.yml target file information
type Target struct {
	Name     string
	Type     string
	ByTarget string `yaml:"by_target"`
	Packager Packager
}

// Packager make.yml package information
type Packager struct {
	Target string
	Option string
}

// StringList make.yml string list('- list: ...')
type StringList struct {
	// Target selector
	Target string
	types  []PlatformID
	items  map[string](*[]string)
}

// Types retrieves list of target types.
func (s *StringList) Types() []PlatformID {
	return s.types
}

// MatchType checks `t` is one of the target type of not.
func (s *StringList) MatchType(t string) bool {
	if len(s.types) == 0 {
		return true // Wildcard
	}
	for _, v := range s.types {
		if v.String() == t {
			return true
		}
	}
	return false
}

// Match checks build conditions are match or not.
func (s *StringList) Match(target string, targetType string) bool {
	if len(s.Target) == 0 || s.Target == target {
		if s.MatchType(targetType) {
			return true
		}
	}
	return false
}

// Items retrieves list associated to `key`.
// Key should be a `string` or can convert to `string` (via .String() method)
// Returns pointer to underlying array (for modifying contents).
func (s *StringList) Items(key interface{}) *[]string {
	if k, ok := key.(string); ok {
		return s.getItems(k)
	}
	if k, ok := key.(fmt.Stringer); ok {
		return s.getItems(k.String())
	}
	return nil
}

func (s *StringList) getItems(key string) *[]string {
	if v, ok := s.items[key]; ok {
		if v != nil {
			return v
		}
	}
	return nil
}

// UnmarshalYAML is the custom handler for mapping YAML to `StringList`
func (s *StringList) UnmarshalYAML(unmarshaler func(interface{}) error) error {
	var fixedSlot struct {
		Types  interface{} `yaml:"type"`
		Target string
	}
	err := unmarshaler(&fixedSlot)
	if err != nil {
		return errors.Wrapf(err, "unmarshaling failed on `StringList` fixed slots")
	}
	var items map[string]*[]string
	err = unmarshaler(&items)
	if err != nil {
		if _, ok := err.(*yaml.TypeError); !ok {
			return err
		}
	}
	switch v := fixedSlot.Types.(type) {
	case string:
		s.types = []PlatformID{PlatformID(v)}
	case []interface{}:
		for _, t := range v {
			item, ok := t.(string)
			if !ok {
				return errors.Errorf("Unexpected item type %v", t)
			}
			s.types = append(s.types, PlatformID(item))
		}
	case nil:
		/*NO-OP*/
	default:
		panic(fmt.Sprintf("type: %v", v))
	}
	s.Target = fixedSlot.Target
	s.items = items
	return nil
}

// Variable make.yml variable section
type Variable struct {
	Name      string
	Value     string
	platforms map[PlatformID]bool
	Target    string
	Build     string
}

// Equals checks v == other.
func (v *Variable) Equals(other *Variable) bool {
	if v.Name != other.Name ||
		v.Value != other.Value ||
		v.Target != other.Target ||
		v.Build != other.Build ||
		len(v.platforms) != len(other.platforms) {
		return false
	}
	for key, val := range v.platforms {
		if oVal, ok := other.platforms[key]; !ok || val != oVal {
			return false
		}
	}
	return true
}

// Platforms retrieves associated PlatformIDs.
func (v *Variable) Platforms() []PlatformID {
	result := make([]PlatformID, 0, len(v.platforms))
	for key := range v.platforms {
		result = append(result, PlatformID(key))
	}
	sort.Slice(result, func(i int, j int) bool {
		return result[i].String() < result[j].String()
	})
	return result
}

// MarshalYAML is called while yaml.Marshal (for custom marshaling).
func (v *Variable) MarshalYAML() (interface{}, error) {
	var result struct {
		Name      string
		Value     string
		Target    string
		Build     string
		Platforms interface{} `yaml:"type,flow,omitempty"`
	}
	result.Name = v.Name
	result.Value = v.Value
	result.Target = v.Target
	result.Build = v.Build
	p := make([]string, 0, len(v.platforms))
	for key := range v.platforms {
		p = append(p, string(key))
	}
	sort.Strings(p)
	switch len(p) {
	case 0:
		result.Platforms = nil
	case 1:
		result.Platforms = p[0]
	default:
		result.Platforms = p
	}
	return result, nil
}

// UnmarshalYAML is called while yaml.Unmarshal (for custom unmarshaing).
func (v *Variable) UnmarshalYAML(unmarshaler func(interface{}) error) error {
	var err error

	var slots struct {
		Name      string
		Value     string
		Target    string
		Build     string
		Platforms interface{} `yaml:"type"`
	}
	err = unmarshaler(&slots)
	if err != nil {
		return errors.Wrapf(err, "unmarshaling failed on `Variable` fixed slots")
	}
	v.Name = slots.Name
	v.Value = slots.Value
	v.Target = slots.Target
	v.Build = slots.Build
	v.platforms = make(map[PlatformID]bool)
	switch val := slots.Platforms.(type) {
	case string:
		v.platforms[PlatformID(val)] = true
	case []interface{}:
		for _, t := range val {
			v.platforms[PlatformID(t.(string))] = true
		}
	case nil:
		/* NO-OP */
	default:
		panic(fmt.Sprintf("type: %v", v))
	}
	return nil
}

// MatchPlatform checks supplied platform `s` is matched or not.
func (v *Variable) MatchPlatform(platform string) bool {
	if len(v.platforms) == 0 {
		return true
	}
	if _, ok := v.platforms[PlatformID(platform)]; ok {
		return true
	}
	return false
}

// GetMatchedValue returns the value of this variable if conditions met.
func (v *Variable) GetMatchedValue(target string, platform string, variant string) (result string, ok bool) {
	if !v.MatchPlatform(platform) {
		return
	}
	if 0 < len(v.Target) && v.Target != target {
		return
	}
	if 0 < len(v.Build) {
		bld := strings.ToLower(v.Build)
		switch variant {
		case Debug.String():
			if bld != "debug" {
				return
			}
		case Release.String():
			if bld != "release" {
				return
			}
		case Develop.String():
			if bld != "develop" {
				return
			}
		case DevelopRelease.String():
			if bld != "develop_release" {
				return
			}
		case Product.String():
			if bld != "product" {
				return
			}
		}
	}
	return v.Value, true
}

// Build in directory source list
// Build describes
type Build struct {
	Name    string
	Command string
	Target  string
	Type    PlatformID
	Deps    string
	Source  []StringList `yaml:",flow"`
}

// Match returns `true` if build target and target-type matched.
func (b *Build) Match(target string, targetType string) bool {
	return (len(b.Target) == 0 || b.Target == target) && (len(b.Type) == 0 || b.Type.String() == targetType)
}

// MatchType checks `platform` is in the target-types
func (b *Build) MatchType(platform string) bool {
	return len(b.Type) == 0 || b.Type.String() == platform
}

// Other make.yml other section
type Other struct {
	Ext         string
	Command     string
	Description string
	NeedDepend  bool `yaml:"need_depend"`
	Type        PlatformID
	Option      []StringList `yaml:",flow"`
}
