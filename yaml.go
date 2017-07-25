// Schema definitions for values from `make.yml`.
package main

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

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
	Target    string
	Platforms *PlatformIDSet
	items     map[string](*[]string)
}

// Types retrieves list of target platforms.
func (s *StringList) Types() []PlatformID {
	if s.Platforms == nil {
		return make([]PlatformID, 0)
	}
	return s.Platforms.ToSlice()
}

// MatchPlatform checks t is in the platform set or not.
func (s *StringList) MatchPlatform(platform string) bool {
	if s.Platforms == nil {
		return true // Wildcard
	}
	return s.Platforms.Contains(platform)
}

// Match checks build conditions are match or not.
func (s *StringList) Match(target string, platform string) bool {
	if len(s.Target) == 0 || s.Target == target {
		if s.MatchPlatform(platform) {
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

// GetMatchedItems retrieves items matched conditions.
func (s *StringList) GetMatchedItems(buildTarget string, platform string, variant string) []string {
	if !s.Match(buildTarget, platform) {
		return nil
	}
	result := make([]string, 0)
	appender := func(key interface{}) {
		if l := s.Items(key); l != nil {
			result = append(result, *l...)
		}
	}
	appender(Common)
	switch variant {
	case Debug.String():
		appender(Debug)
	case Develop.String():
		appender(Develop)
	case Release.String():
		appender(Release)
	case DevelopRelease.String():
		appender(DevelopRelease)
	case Product.String():
		appender(Product)
	default:
		/* NO-OP */
	}
	return result
}

// UnmarshalYAML is the custom handler for mapping YAML to `StringList`
func (s *StringList) UnmarshalYAML(unmarshaler func(interface{}) error) error {
	var fixedSlot struct {
		Types  *PlatformIDSet `yaml:"type"`
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
	s.Platforms = fixedSlot.Types
	s.Target = fixedSlot.Target
	s.items = items
	return nil
}

// Variable make.yml variable section
type Variable struct {
	Name      string
	Value     string
	Platforms *PlatformIDSet `yaml:"type"`
	Target    string
	Build     string
}

// Equals checks v == other.
func (v *Variable) Equals(other *Variable) bool {
	if !(v.Name == other.Name &&
		v.Value == other.Value &&
		v.Target == other.Target &&
		v.Build == other.Build) {
		return false
	}
	if v.Platforms == nil {
		if other.Platforms == nil {
			return true
		}
		return false
	}
	return v.Platforms.Equals(*other.Platforms)
}

// MatchPlatform checks supplied platform `s` is matched or not.
func (v *Variable) MatchPlatform(platform string) bool {
	if v.Platforms == nil {
		return true
	}
	return v.Platforms.Contains(platform)
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

//MatchType checks `platform` is in the target platforms.
func (b *Build) MatchType(platform string) bool {
	return len(b.Type) == 0 || b.Type.String() == platform
}

// Other make.yml other section
type Other struct {
	Extension   string         `yaml:"ext"`
	Command     string         `yaml:"command"`
	Description string         `yaml:"description"`
	NeedDepend  bool           `yaml:"need_depend"`
	Platforms   *PlatformIDSet `yaml:"type"`
	Option      []StringList   `yaml:"option,flow"`
}

// MatchPlatform checks `platform` is in the set or not.
func (o *Other) MatchPlatform(platform string) bool {
	if o.Platforms == nil {
		return true
	}
	return o.Platforms.Contains(platform)
}
