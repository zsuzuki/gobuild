package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"errors"
)

// BuildInfo is build information in directory
type BuildInfo struct {
	variables      map[string]string
	includes       []string
	defines        []string
	options        []string
	archiveOptions []string
	convertOptions []string
	linkOptions    []string
	linkDepends    []string
	libraries      []string
	packageTarget  string
	packageCommand string
	selectTarget   string
	target         string
	outputdir      string
	subdir         []string
	mydir          string
	tests          []string
}

// Retrieves command line option prefix
func (info BuildInfo) OptionPrefix() string {
	if pfx, exists := info.variables["option_prefix"]; exists {
		return pfx
	}
	return "-"
}

// Appends include path.
func (info *BuildInfo) AddInclude(path string) {
	pfx := info.OptionPrefix()
	if p := filepath.ToSlash(filepath.Clean(path)); strings.Index(p, " ") != -1 {
		info.includes = append(info.includes, fmt.Sprintf("\"%sI%s\"", pfx, p))
	} else {
		info.includes = append(info.includes, fmt.Sprintf("%sI%s", pfx, p))
	}
}

// Appends macro definitions.
func (info *BuildInfo) AddDefines(def string) {
	pfx := info.OptionPrefix()
	info.defines = append(info.defines, fmt.Sprintf("%sD%s", pfx, def))
}

// Interpolates given string `s`.
// Note: Handles $out, $in...
func (info *BuildInfo) Interpolate (s string) (string, error){
	if idx := strings.Index(s, "${"); 0 <= idx {
		expanded, err := Interpolate(s[idx:], info.variables)
		if err != nil {
			return "", err
		}
		return s[:idx] + expanded, nil
	}
	return s, nil
}

// Strictly interpolates given string `s`.
// Note: Handles $out, $in...
func (info *BuildInfo) StrictInterpolate (s string) (string, error){
	return Interpolate(s, info.variables)
}

// Retrieves the value associated to symbol `s`.
func (info *BuildInfo) ExpandVariable (s string) (string, error) {
	if str, exists := info.variables [s]; exists {
		return info.Interpolate(str)
	}
	return "", errors.New (fmt.Sprintf ("Variable \"%s\" was not defined.", s))
}