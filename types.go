package main

import (
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// KnownBuildType represents the `known` build type
type KnownBuildType string

const (
	// Common represents common definitions for all build types.
	Common KnownBuildType = "list"
	// Debug represents definitions for the `debug` build.
	Debug KnownBuildType = "debug"
	// Release represents definitions for the `release` build.
	Release KnownBuildType = "release"
	// Develop represents definitions for the `develop` build.
	Develop KnownBuildType = "develop"
	// DevelopRelease represents definitions for the `develop-release` build (i.e. beta).
	DevelopRelease KnownBuildType = "develop-release"
	// Product represents definitions for the `product` build (for shipping).
	Product KnownBuildType = "product"
)

// KnownBuildTypes lists all of `known` build types.
var KnownBuildTypes = [...]KnownBuildType{
	Common,
	Debug,
	Release,
	Develop,
	DevelopRelease,
	Product}

// String returns the string representation of the `KnownBuildType`
func (t KnownBuildType) String() string {
	return string(t)
}

// StringList make.yml string list('- list: ...')
type StringList struct {
	Type   string
	Target string
	items  map[string](*[]string)
}

// Match checks build conditions are match or not.
func (s *StringList) Match(target string, targetType string) bool {
	return (s.Target == "" || s.Target == target) && (s.Type == "" || s.Type == targetType)
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
		Type   string
		Target string
	}
	err := unmarshaler(&fixedSlot)
	if err != nil {
		return errors.Wrapf(err, "Unmarshaling failed on `StringList` fixed slot")
	}
	var items map[string]*[]string
	err = unmarshaler(&items)
	if err != nil {
		if _, ok := err.(*yaml.TypeError); !ok {
			return err
		}
	}
	s.Type = fixedSlot.Type
	s.Target = fixedSlot.Target
	s.items = items
	return nil
}

// Packager make.yml package information
type Packager struct {
	Target string
	Option string
}

// Target make.yml target file information
type Target struct {
	Name     string
	Type     string
	ByTarget string `yaml:"by_target"`
	Packager Packager
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
	ConvertList   []StringList `yaml:"convert_list,flow"`
	Subdirs       []StringList `yaml:"subdir,flow"`
	Tests         []StringList `yaml:",flow"`
	Other         []Other      `yaml:",flow"`
	SubNinja      []StringList `yaml:",flow"`
}

//
// build information
//

// OtherRule is used for building non-default targets (ex. .bin .dat ...)
type OtherRule struct {
	Compiler    string
	Command     string
	Title       string
	Options     []string
	needInclude bool
	needOption  bool
	needDefine  bool
	NeedDepend  bool
}

// OtherRuleFile is not default target(ex. .bin .dat ...) file information
type OtherRuleFile struct {
	Rule     string
	Compiler string
	Infile   string
	Outfile  string
	Include  string
	Option   string
	Define   string
	Depend   string
}

// AppendBuild ...
type AppendBuild struct {
	Command string
	Desc    string
	Deps    bool
}

// BuildCommand is command set for one target file
type BuildCommand struct {
	Command          string
	CommandType      string
	CommandAlias     string
	Args             []string
	InFiles          []string
	OutFile          string
	DepFile          string
	Depends          []string
	ImplicitDepends  []string
	NeedCommandAlias bool
}
