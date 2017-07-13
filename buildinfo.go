package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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
	selectedTarget string // Target explicitly specified via command-line.
	target         string
	outputdir      string
	subdir         []string
	mydir          string
	tests          []string
}

// OptionPrefix retrieves command line option prefix
func (info *BuildInfo) OptionPrefix() string {
	if pfx, exists := info.variables["option_prefix"]; exists {
		return pfx
	}
	return "-"
}

// AddInclude appends include path.
func (info *BuildInfo) AddInclude(path string) {
	pfx := info.OptionPrefix()
	if p := filepath.ToSlash(filepath.Clean(path)); strings.Index(p, " ") != -1 {
		info.includes = append(info.includes, fmt.Sprintf("\"%sI%s\"", pfx, p))
	} else {
		info.includes = append(info.includes, fmt.Sprintf("%sI%s", pfx, p))
	}
}

// AddDefines appends macro definitions.
// Assumes input is `KEY` or `KEY=VALUE`.
// If `KEY` contains '-', replace it to '_'
func (info *BuildInfo) AddDefines(def string) {
	idef, err := info.StrictInterpolate(def)
	if err != nil {
		Warn("not found variable in <%s>\n", def)
		return // Should be handled properly
	}
	info.defines = append(info.defines, (func(s string) string {
		kv := strings.SplitN(s, "=", 2)
		switch len(kv) {
		case 0: // WHAT?
			return s
		case 1:
			return fmt.Sprintf("%sD%s",
				info.OptionPrefix(),
				strings.Replace(kv[0], "-", "_", -1))
		case 2:
			fallthrough
		default:
			return fmt.Sprintf("%sD%s=%s",
				info.OptionPrefix(),
				strings.Replace(kv[0], "-", "_", -1),
				kv[1])
		}
	})(idef))
}

// Interpolate interpolates given string `s`.
// Note: Handles $out, $in...
func (info *BuildInfo) Interpolate(s string) (string, error) {
	if idx := strings.Index(s, "${"); 0 <= idx {
		expanded, err := Interpolate(s[idx:], info.variables)
		if err != nil {
			return "", err
		}
		return s[:idx] + expanded, nil
	}
	return s, nil
}

// StrictInterpolate strictly interpolates given string `s`.
// Note: Handles $out, $in...
func (info *BuildInfo) StrictInterpolate(s string) (string, error) {
	if idx := strings.Index(s, "${"); 0 <= idx {
		expanded, err := StrictInterpolate(s[idx:], info.variables)
		if err != nil {
			return "", err
		}
		return s[:idx] + expanded, nil
	}
	return s, nil
}

// ExpandVariable retrieves the value associated to symbol `s`.
func (info *BuildInfo) ExpandVariable(s string) (string, error) {
	if str, exists := info.variables[s]; exists {
		return info.Interpolate(str)
	}
	return "", errors.Errorf("variable \"%s\" is not defined", s)
}

// MakeExecutablePath constructs path for the executables in platform dependent way.
func (info *BuildInfo) MakeExecutablePath(s string) string {
	if suffix, ok := info.variables["execute_suffix"]; ok {
		return filepath.Join(info.outputdir, s+suffix)
	}
	return filepath.Join(info.outputdir, s)
}
