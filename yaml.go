// Schema definitions for values from `make.yml`.
package main

import (
	"fmt"

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
	types  []string
	Target string
	items  map[string](*[]string)
}

// Types retrieves list of target types.
func (s *StringList) Types() []string {
	return s.types
}

// MatchType checks `t` is one of the target type of not.
func (s *StringList) MatchType(t string) bool {
	if len(s.types) == 0 {
		return true // Wildcard
	}
	for _, v := range s.types {
		if v == t {
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
		return errors.Wrapf(err, "unmarshaling failed on `StringList` fixed slot")
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
		s.types = []string{v}
	case []interface{}:
		for _, t := range v {
			s.types = append(s.types, t.(string))
		}
	case nil:
		/*NO-OP*/
	default:
		panic(fmt.Sprintf("type: %v", v))
	}
	//s.types = fixedSlot.Type
	s.Target = fixedSlot.Target
	s.items = items
	return nil
}

// Variable make.yml variable section
type Variable struct {
	Name   string
	Value  string
	Type   string
	Target string
	Build  string
}

// Build in directory source list
type Build struct {
	Name    string
	Command string
	Target  string
	Type    string
	Deps    string
	Source  []StringList `yaml:",flow"`
}

// Match returns `true` if build target and target-type matched.
func (b *Build) Match(target string, targetType string) bool {
	return (b.Target == "" || b.Target == target) && (b.Type == "" || b.Type == targetType)
}

// Other make.yml other section
type Other struct {
	Ext         string
	Command     string
	Description string
	NeedDepend  bool `yaml:"need_depend"`
	Type        string
	Option      []StringList `yaml:",flow"`
}
