package main

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
	Product,
}

// String returns the string representation of the `KnownBuildType`
func (t KnownBuildType) String() string {
	return string(t)
}

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
	Project  string
}

// AppendBuild ...
type AppendBuild struct {
	Command string
	Desc    string
	Deps    bool
}

func (a *AppendBuild) Equals(b *AppendBuild) bool {
	return a.Command == b.Command && a.Desc == b.Desc && a.Deps == b.Deps
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
	Project          string
}
