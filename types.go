package main

// Packager make.yml package information
type Packager struct {
	Target string
	Option string
}

// Target make.yml target file information
type Target struct {
	Name      string
	Type      string
	By_Target string
	Packager  Packager
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

// Other make.yml other section
type Other struct {
	Ext           string
	Command       string
	Description   string
	needDependend bool
	Type          string
	Option        []StringList `yaml:",flow"`
}

// Data format make.yml top structure
type Data struct {
	Target         []Target     `yaml:",flow"`
	Include        []StringList `yaml:",flow"`
	Variable       []Variable   `yaml:",flow"`
	Define         []StringList `yaml:",flow"`
	Option         []StringList `yaml:",flow"`
	Archive_Option []StringList `yaml:",flow"`
	Convert_Option []StringList `yaml:",flow"`
	Link_Option    []StringList `yaml:",flow"`
	Link_Depend    []StringList `yaml:",flow"`
	Libraries      []StringList `yaml:",flow"`
	Prebuild       []Build      `yaml:",flow"`
	Postbuild      []Build      `yaml:",flow"`
	Source         []StringList `yaml:",flow"`
	Convert_List   []StringList `yaml:",flow"`
	Subdir         []StringList `yaml:",flow"`
	Tests          []StringList `yaml:",flow"`
	Other          []Other      `yaml:",flow"`
	SubNinja       []StringList `yaml:",flow"`
}

//
// error
//

// MyError is override error in application
type MyError struct {
	str string
}

func (m MyError) Error() string {
	return m.str
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
	rule     string
	compiler string
	infile   string
	outfile  string
	include  string
	option   string
	define   string
	depend   string
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
	NeedCommandAlias bool
}

// BuildResult is result by build(in directory)
type BuildResult struct {
	success    bool
	createList []string
}
