package main

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

// StringList make.yml string list('- list: ...')
type StringList struct {
	Type           string
	Target         string
	Debug          []string `yaml:",flow"`
	Release        []string `yaml:",flow"`
	Develop        []string `yaml:",flow"`
	DevelopRelease []string `yaml:",flow"`
	Product        []string `yaml:",flow"`
	List           []string `yaml:",flow"`
}

// Checks build conditions are match or not.
func (s *StringList) Match(target string, targetType string) bool {
	return (s.Target == "" || s.Target == target) && (s.Type == "" || s.Type == targetType)
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

// Returns `true` if build target and target-type matched.
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
