//
//
//
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/kuma777/go-msbuild"
)

//
// global variables
//
const (
	cbuildVersion  = "1.0.2"
	buildDirectory = "CBuild.dir"
)

var (
	option struct {
		targetType string
		targetName string
		outputDir  string
		verbose    bool
		ninjaFile  string
		variant    string
	}

	useResponse     bool
	groupArchives   bool
	responseNewline bool

	subNinjaList      []string
	appendRules       map[string]AppendBuild
	otherRuleList     map[string]OtherRule
	commandList       []BuildCommand
	otherRuleFileList []OtherRuleFile

	// ScannedConfigs remembers all scanned configuration files.
	ScannedConfigs []string

	// ProgramName holds invoked program name.
	ProgramName = getExeName()

	rxTruthy = regexp.MustCompile(`^\s*(?i:t(?:rue)?|y(?:es)?|on|1)(?:\s+.*)?$`)
	rxFalsy  = regexp.MustCompile(`^\s*(?i:f(?:alse)|no?|off|0)(?:\s+.*)?$`)
)

// The entry point.
func main() {

	//ProgramName := getExeName()

	genMSBuild := false
	projdir := ""
	projname := ""
	var (
		isDebug          bool
		isRelease        bool
		isProduct        bool
		isDevelop        bool
		isDevelopRelease bool
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [<target>]\n", ProgramName)
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.BoolVar(&option.verbose, "v", false, "verbose mode")
	flag.BoolVar(&isRelease, "release", false, "release build")
	flag.BoolVar(&isDebug, "debug", true, "debug build")
	flag.BoolVar(&isDevelop, "develop", false, "develop(beta) build")
	flag.BoolVar(&isDevelopRelease, "develop_release", false, "develop(beta) release build")
	flag.BoolVar(&isProduct, "product", false, "for production build")
	flag.StringVar(&option.targetType, "type", "default", "build target type")
	flag.StringVar(&option.targetName, "t", "", "build target name")
	flag.StringVar(&option.outputDir, "o", "build", "build directory")
	flag.StringVar(&option.ninjaFile, "f", "build.ninja", "output build.ninja filename")
	flag.BoolVar(&genMSBuild, "msbuild", false, "Export MSBuild project")
	flag.StringVar(&projdir, "msbuild-dir", "./", "MSBuild project output directory")
	flag.StringVar(&projname, "msbuild-proj", "out", "MSBuild project name")
	showVersionAndExit := flag.Bool("version", false, "display version")
	flag.Parse()

	if *showVersionAndExit {
		fmt.Fprintf(os.Stdout, "%s: %v (%s/%s)\n", ProgramName, cbuildVersion, runtime.Version(), runtime.Compiler)
		os.Exit(0)
	}

	option.variant = Debug.String()
	if isDebug {
		option.variant = Debug.String()
	}
	if isProduct {
		option.variant = Product.String()
	}
	if isRelease {
		option.variant = Release.String()
	}
	if isDevelopRelease {
		option.variant = DevelopRelease.String()
	}
	if isDevelop {
		option.variant = Develop.String()
	}

	useResponse = false
	groupArchives = false
	responseNewline = false

	if 0 < flag.NArg() && option.targetName == "" {
		option.targetName = flag.Arg(0)
	}

	appendRules = map[string]AppendBuild{}
	commandList = []BuildCommand{}
	otherRuleList = map[string]OtherRule{}
	otherRuleFileList = []OtherRuleFile{}
	subNinjaList = []string{}

	if option.targetName != "" {
		Verbose("%s: Target is \"%s\"\n", ProgramName, option.targetName)
	}
	buildinfo := BuildInfo{
		variables:      map[string]string{"option_prefix": "-"},
		includes:       []string{},
		defines:        []string{},
		options:        []string{},
		archiveOptions: []string{},
		convertOptions: []string{},
		linkOptions:    []string{},
		selectTarget:   option.targetName,
		target:         option.targetName}

	if _, err := CollectConfigurations(buildinfo, ""); err != nil {
		fmt.Fprintf(os.Stderr, "%s:error: %v\n", ProgramName, err)
		os.Exit(1)
	}
	if nlen := len(commandList) + len(otherRuleFileList); nlen <= 0 {
		fmt.Fprintf(os.Stderr, "%s: No commands to run.\n", ProgramName)
		os.Exit(0)
	}
	if genMSBuild {
		Verbose("%s: Creates VC++ project file(s).\n", ProgramName)
		outputMSBuild(projdir, projname)
	} else {
		if err := outputNinja(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v", ProgramName, err)
			os.Exit(1)
		}
	}
	os.Exit(0)
}

// Obtains executable name if possible.
func getExeName() string {
	var name = "gobuild"
	if n, err := os.Executable(); err == nil {
		name = n
	}
	return filepath.ToSlash(name)
}

// CollectConfigurations collects configurations recursively.
func CollectConfigurations(info BuildInfo, relChildDir string) (*[]string, error) {
	var childPath string
	if relChildDir == "" {
		childPath = "./"
	} else {
		childPath = filepath.ToSlash(filepath.Clean(relChildDir)) + "/"
	}
	return traverse(info, childPath, 0)
}

func traverse(info BuildInfo, relChildDir string, level int) (*[]string, error) {
	// result.Successed = false // Named return value is initialized to 0
	if relChildDir[len(relChildDir)-1] != '/' {
		return nil, errors.New("output directory should end with '/'")
	}
	Verbose("%s: Enter \"%s\"\n", ProgramName, relChildDir)
	defer Verbose("%s: Leave \"%s\"\n", ProgramName, relChildDir)

	yamlSource := filepath.Join(relChildDir, "make.yml")
	buf, err := ioutil.ReadFile(yamlSource)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read \"%s\"", yamlSource)
	}

	var d Data
	err = yaml.Unmarshal(buf, &d)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal \"%s\"", yamlSource)
	}
	ScannedConfigs = append(ScannedConfigs, yamlSource)

	info.mydir = relChildDir
	//
	// select target
	//
	currentTarget, _, ok := getTarget(info, d.Target)
	if !ok {
		return nil, errors.New("no targets")
	}
	if info.target == "" {
		info.target = currentTarget.Name
		Verbose("%s: Target is \"%s\".\n", ProgramName, info.target)
	}
	info.selectTarget = ""

	if level == 0 && option.targetType == "default" {
		option.targetType = checkType(d.Variable)
	}

	// Merge variable definitions (parent + current).
	newvar := map[string]string{}
	for ik, iv := range info.variables {
		newvar[ik] = iv
	}
	info.variables = newvar
	for _, v := range d.Variable {
		if val, ok := v.getValue(&info); ok {
			switch v.Name {
			case "enable_response":
				useResponse = ToBoolean(val)
			case "response_newline":
				responseNewline = ToBoolean(val)
			case "group_archives":
				groupArchives = ToBoolean(val)
			default: /* NO-OP */
			}
			info.variables[v.Name] = val
		}
	}
	optionPrefix := info.OptionPrefix()

	if level == 0 {
		switch option.variant {
		case Product.String():
			option.outputDir = JoinPathes(option.outputDir, option.targetType, "Product")
		case Develop.String():
			option.outputDir = JoinPathes(option.outputDir, option.targetType, "Develop")
		case DevelopRelease.String():
			option.outputDir = JoinPathes(option.outputDir, option.targetType, "DevelopRelease")
		case Release.String():
			option.outputDir = JoinPathes(option.outputDir, option.targetType, "Release")
		default:
			option.outputDir = JoinPathes(option.outputDir, option.targetType, "Debug")
		}
	}

	info.outputdir = JoinPathes(option.outputDir, relChildDir) + "/" // Proofs '/' ending

	// Constructs include path arguments.
	for _, pth := range getList(d.Include, info.target) {
		const prefix = "$output"
		if strings.HasPrefix(pth, prefix) {
			info.AddInclude(JoinPathes(info.outputdir, "output"+pth[len(prefix):]))
		} else {
			useRel := pth[0] == '$'
			pth, err = info.StrictInterpolate(pth)
			if err != nil {
				return nil, err
			}
			if !useRel && !filepath.IsAbs(pth) {
				info.AddInclude(JoinPathes(relChildDir, pth))
			} else {
				info.AddInclude(pth)
			}
		}
	}
	// Constructs defines.
	for _, d := range getList(d.Define, info.target) {
		info.AddDefines(d)
	}
	// Construct other options.
	for _, o := range getList(d.Option, info.target) {
		info.options, err = appendOption(info, info.options, o, optionPrefix)
		if err != nil {
			return nil, err
		}
	}
	// Constructs option list for archiver.
	for _, a := range getList(d.ArchiveOption, info.target) {
		info.archiveOptions, err = appendOption(info, info.archiveOptions, a, "")
		if err != nil {
			return nil, err
		}
	}
	// Constructs option list for converters.
	for _, c := range getList(d.ConvertOption, info.target) {
		info.convertOptions, err = appendOption(info, info.convertOptions, c, "")
		if err != nil {
			return nil, err
		}
	}
	// Construct option list for linker.
	for _, l := range getList(d.LinkOption, info.target) {
		info.linkOptions, err = appendOption(info, info.linkOptions, l, optionPrefix)
		if err != nil {
			return nil, err
		}
	}
	// Constructs system library list.
	for _, ls := range getList(d.Libraries, info.target) {
		info.libraries, err = appendOption(info, info.libraries, ls, optionPrefix+"l")
		if err != nil {
			return nil, err
		}
	}
	// Constructs library list.
	for _, ld := range getList(d.LinkDepend, info.target) {
		info.linkDepends, err = appendOption(info, info.linkDepends, ld, "")
		if err != nil {
			return nil, err
		}
	}
	// Constructs sub-ninjas
	for _, subninja := range getList(d.SubNinja, info.target) {
		subNinjaList = append(subNinjaList, subninja)
	}

	if err = createOtherRule(info, d.Other, optionPrefix); err != nil {
		return nil, err
	}

	files := getList(d.Source, info.target)
	cvfiles := getList(d.ConvertList, info.target)
	testfiles := getList(d.Tests, info.target)

	// sub-directories
	subdirs := getList(d.Subdirs, info.target)

	subArtifacts := []string{}

	// Recurse into the sub-directories.
	for _, s := range subdirs {
		// relChildDir always ends with '/'
		odir := relChildDir + filepath.ToSlash(filepath.Clean(s)) + "/"
		if r, err := traverse(info, odir, level+1); err == nil {
			if 0 < len(*r) {
				subArtifacts = append(subArtifacts, *r...)
			}
		} else {
			return nil, err
		}
	}

	// pre build files
	if err = createPrebuild(info, relChildDir, d.Prebuild); err != nil {
		return nil, err
	}

	// create compile list
	artifacts, err := compileFiles(info, relChildDir, files)
	if err != nil {
		return nil, err
	}

	var result []string

	switch currentTarget.Type {
	case "library":
		// archive
		if 0 < len(artifacts) {
			// MEMO: Constructs relation
			//   <lib> 1--0..* <artifacts>
			libName, err := createArchive(info, artifacts, currentTarget.Name)
			if err != nil {
				return nil, err
			}
			result = append(subArtifacts, libName)
		} else {
			Warn("There are no files to build in \"%s\".", relChildDir)
		}
	case "execute":
		// link program
		if 0 < len(artifacts) || 0 < len(subArtifacts) {
			// MEMO: Constructs relation
			//   <exe> 1--1..* <artifacts>
			//     1\
			//       +-- 1..* <artifacts from sub-directories>
			err = createLink(
				info,
				append(artifacts, subArtifacts...),
				currentTarget.Name,
				currentTarget.Packager)
			if err != nil {
				return nil, err
			}
		} else {
			Warn("There are no files to build in \"%s\".", relChildDir)
		}
	case "convert":
		if 0 < len(cvfiles) {
			cmd, e := makeConvertCommand(info, relChildDir, cvfiles, currentTarget.Name)
			if e != nil {
				return nil, e
			}
			commandList = append (commandList, *cmd)
		} else {
			Warn("There are no files to convert in \"%s\".", relChildDir)
		}
	case "passthrough":
		// Just bubbling up the artifacts
		result = append(subArtifacts, artifacts...)
	case "test":
		// unit tests
		if e := createTest(info, testfiles, relChildDir); e != nil {
			return nil, e
		}
	default:
		/* NO-OP */
	}

	Verbose("%s: Artifacts in \"%s\":\n", ProgramName, relChildDir)
	if option.verbose {
		for _, rc := range result {
			fmt.Fprintf(os.Stderr, "#   %s\n", rc)
		}
	}
	return &result, nil
}

//
// Retrieves items associated to `targetName`.
//
func getList(block []StringList, targetName string) []string {
	lists := []string{}
	for _, item := range block {
		if item.Match(targetName, option.targetType) {
			appender := func(key interface{}) {
				if l := item.Items(key); l != nil {
					lists = append(lists, *l...)
				}
			}
			appender(Common)
			switch option.variant {
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
		}
	}
	return lists
}

func stringToReplacedList(info BuildInfo, str string) ([]string, error) {
	sl := strings.Split(str, " ")
	for i, s := range sl {
		expanded, err := info.Interpolate(s)
		if err != nil {
			return []string{}, err
		}
		sl[i] = expanded
	}
	return sl, nil
}

//
// archive objects
//
func createArchive(info BuildInfo, inputs []string, targetName string) (string, error) {
	arCommand, e := info.ExpandVariable("archiver")
	if e != nil {
		return "", e
	}

	cmd := BuildCommand{
		Command:     arCommand,
		CommandType: "ar",
		Args:        info.archiveOptions,
		InFiles:     inputs,
		OutFile: (func() string {
			switch option.targetType {
			case "WIN32":
				return JoinPathes(info.outputdir, targetName+".lib")
			default:
				return JoinPathes(info.outputdir, fmt.Sprintf("lib%s.a", targetName))
			}
		})(),
		NeedCommandAlias: true}
	commandList = append(commandList, cmd) // Record command

	return cmd.OutFile, nil
}

//
// link objects
//
func createLink(info BuildInfo, createList []string, targetName string, packager Packager) error {
	targetPath := JoinPathes(info.MakeExecutablePath(targetName))

	linker, e := info.ExpandVariable("linker")
	if e != nil {
		return e
	}

	options := []string{}
	for _, lo := range info.linkOptions {
		lo = strings.Replace(lo, "$out", targetPath, -1)
		options = append(options, lo)
	}
	options = append(options, info.libraries...)

	// execute
	cmd := BuildCommand{
		Command:          linker,
		CommandType:      "link",
		Args:             options,
		InFiles:          createList,
		OutFile:          targetPath,
		Depends:          info.linkDepends,
		NeedCommandAlias: true}
	commandList = append(commandList, cmd)
	//fmt.Println("-o " + NowTarget.Name + flist)

	if 0 < len(packager.Target) {
		// package
		pkgname := JoinPathes(option.outputDir, targetName, packager.Target)
		pkgr, e := info.ExpandVariable("packager")
		if e != nil {
			return e
		}
		pargs, e := stringToReplacedList(info, packager.Option)
		if e != nil {
			return e
		}
		pkg := BuildCommand{
			Command:          pkgr,
			CommandType:      "packager",
			Args:             pargs,
			InFiles:          []string{targetPath},
			OutFile:          pkgname,
			NeedCommandAlias: true}
		commandList = append(commandList, pkg) // Record commands
	}
	return nil
}

//
// convert objects
//
func makeConvertCommand(info BuildInfo, loaddir string, createList []string, targetName string) (*BuildCommand, error) {
	cvname := JoinPathes(info.outputdir, targetName)
	converter, ok := info.variables["converter"]
	if !ok {
		return nil, errors.Errorf("missing the `converter` definitions for \"%s\"", targetName)
	}

	clist := []string{}
	for _, f := range createList {
		clist = append(clist, JoinPathes(loaddir, f))
	}

	cmd := BuildCommand{
		Command:          converter,
		CommandType:      "convert",
		Args:             info.convertOptions,
		InFiles:          clist,
		OutFile:          cvname,
		NeedCommandAlias: true}
	commandList = append(commandList, cmd) // Record commands
	return &cmd, nil
}

//
// unit tests
//
func createTest(info BuildInfo, createList []string, loaddir string) error {
	carg := append(info.includes, info.defines...)
	for _, ca := range info.options {
		switch ca {
		case "$out":
		case "$dep":
		case "$in":
		case "-c":
			/* NO-OP */
		default:
			carg = append(carg, ca)
		}
	}
	//objdir := filepath.Join(info.outputdir, buildDirectory) + "/"
	//objdir := info.outputdir

	for _, f := range createList {
		// first, compile a test driver
		createList, _ = compileFiles(info, loaddir, []string{f})

		// then link it as an executable (test_aaa.cpp -> test_aaa)
		trname := strings.TrimSuffix(f, filepath.Ext(f))

		if esuf, ok := info.variables["execute_suffix"]; ok {
			trname += esuf
		}
		createLink(info, createList, JoinPathes(trname), Packager{})
	}
	return nil
}

//
// option
//
func appendOption(info BuildInfo, lists []string, opt string, optionPrefix string) ([]string, error) {
	for _, so := range strings.Split(optionPrefix+opt, " ") {
		so, err := info.StrictInterpolate(so)
		if err != nil {
			return lists, err
		}
		if strings.Index(so, " ") != -1 {
			so = "\"" + so + "\""
		}
		lists = append(lists, so)
	}
	return lists, nil
}

//
// target
//
func getTarget(info BuildInfo, tlist []Target) (Target, string, bool) {
	if info.selectTarget != "" {
		// search target
		for _, t := range tlist {
			if info.selectTarget == t.Name {
				return t, "_" + info.selectTarget, true
			}
		}

	} else {
		if info.target != "" {

			// search by_target
			for _, t := range tlist {
				if info.target == t.ByTarget {
					return t, "_" + info.target, true
				}
			}
			// search target
			for _, t := range tlist {
				if info.target == t.Name {
					return t, "_" + info.target, true
				}
			}
		}
		if 0 < len(tlist) {
			t := tlist[0]
			if info.target == "" {
				return t, "_" + t.Name, true
			}
			return t, "", true
		}
	}
	return Target{}, "", false
}

// FixupCommandPath fixes command path (appeared at the 1st element).
// Returns fixed command-line and command path
func FixupCommandPath(command string, commandDir string) (commandLine string, commandPath string) {
	args := strings.Split(command, " ")
	cmd := args[0]
	commandPath = JoinPathes(commandDir, cmd)
	args[0] = commandPath
	commandLine = strings.Join(args, " ")
	return
}

//
// compile files
//
func createPrebuild(info BuildInfo, loaddir string, buildItems []Build) error {
	for _, build := range buildItems {
		if !build.Match(info.target, option.targetType) {
			continue
		}

		// register prebuild
		sources := getList(build.Source, info.target)
		if len(sources) == 0 {
			return errors.Errorf("no sources for command `%s`", build.Name)
		}
		{
			// Fixes source elements
			for i, src := range sources {
				if src[0] == '$' {
					sabs, _ := filepath.Abs(filepath.Join(info.outputdir, "output", src[1:]))
					sources[i] = escapeDriveColon(JoinPathes(sabs))
				} else if src == "always" {
					sources[i] = "always |"
				} else {
					expanded, err := info.Interpolate(src)
					if err != nil {
						return err
					}
					sources[i] = JoinPathes(loaddir, expanded)
				}
			}
		}
		buildCommand, ok := info.variables[build.Command]
		if !ok {
			return errors.Errorf("missing build command \"%s\" (referenced from \"%s\")", build.Command, build.Name)
		}
		buildCommand, err := info.Interpolate(strings.Replace(buildCommand, "${selfdir}", loaddir, -1))
		if err != nil {
			return err
		}
		commandLabel := strings.Replace(JoinPathes(info.outputdir, build.Command), "/", "_", -1)
		deps := []string{}

		if _, ok = appendRules[commandLabel]; !ok {
			// Create a rule...
			// Fixes command path.
			if buildCommand[0] == '$' {
				r, d := FixupCommandPath(buildCommand[1:], info.outputdir)
				abs, err := filepath.Abs(d)
				if err != nil {
					return errors.Wrapf(err, "failed to obtain the absolute path for \"%s\"", d)
				}
				d = filepath.ToSlash(abs)
				deps = append(deps, d)
				buildCommand = r
			} else if strings.HasPrefix(buildCommand, "../") || strings.HasPrefix(buildCommand, "./") {
				// Commands are relative to `make.yml`'s directory
				r, d := FixupCommandPath(buildCommand, loaddir)
				deps = append(deps, d)
				buildCommand = r
			}
			useDeps := false
			if build.Deps != "" {
				useDeps = true
			}
			ab := AppendBuild{
				Command: strings.Replace(buildCommand, "$target", info.target, -1),
				Desc:    build.Command,
				Deps:    useDeps}
			appendRules[commandLabel] = ab
		}

		if build.Name[0] != '$' || strings.HasPrefix(build.Name, "$target/") {
			pn := build.Name
			if pn[0] == '$' { // bulid.Name is "$target/..."
				pn = strings.Replace(pn, "$target/", fmt.Sprintf("/.%s/", info.target), 1)
			}
			outfile, _ := filepath.Abs(filepath.Join(info.outputdir, pn))
			outfile = escapeDriveColon(JoinPathes(outfile))
			cmd := BuildCommand{
				Command:          build.Command,
				CommandType:      commandLabel,
				Depends:          deps,
				InFiles:          sources,
				OutFile:          outfile,
				NeedCommandAlias: false}
			commandList = append(commandList, cmd)
		} else {
			// Found `$...`
			ext := build.Name[1:] //filepath.Ext(build.Name)
			for _, src := range sources {
				dst := filepath.Base(src)
				next := filepath.Ext(src)
				if next != "" {
					dst = dst[0:len(dst)-len(next)] + ext
				} else {
					dst += ext
				}
				outfile, _ := filepath.Abs(filepath.Join(info.outputdir, "output", dst))
				outfile = escapeDriveColon(JoinPathes(outfile))
				cmd := BuildCommand{
					Command:          build.Command,
					CommandType:      commandLabel,
					Depends:          deps,
					InFiles:          []string{src},
					OutFile:          outfile,
					NeedCommandAlias: false}
				commandList = append(commandList, cmd)
			}
		}
	}
	return nil
}

// Build command for compiling C, C++...
func compileFiles(info BuildInfo, loaddir string, files []string) (createList []string, e error) {
	if len(files) == 0 {
		return // Nothing to do
	}
	compiler, e := info.ExpandVariable("compiler")
	if e != nil {
		return []string{}, e
	}

	pchFile := createPCH(info, loaddir, compiler)

	arg1 := append(info.includes, info.defines...)

	for _, srcPath := range files {
		dstPathBase := srcPath // `dstPathBase` contains the basename of the `srcPath`.
		var objdir string
		if srcPath[0] == '$' {
			// Auto generated pathes.
			if strings.HasPrefix(srcPath, "$target/") {
				dstPathBase = strings.Replace(dstPathBase, "$target/", fmt.Sprintf("/.%s/", info.target), 1)
			} else {
				dstPathBase = srcPath[1:]
			}
			srcPath = filepath.Join(info.outputdir, dstPathBase)
			dstPathBase = filepath.Base(dstPathBase)
			objdir = JoinPathes(filepath.Dir(srcPath), buildDirectory)
		} else {
			// At this point, `srcPath` is a relative path rooted from `loaddir`
			tf := filepath.Join(loaddir, srcPath)
			objdir = JoinPathes(filepath.Dir(filepath.Join(info.outputdir, srcPath)), buildDirectory)
			dstPathBase = filepath.Base(tf)
			srcPath = tf
		}
		srcPath, _ = filepath.Abs(srcPath)
		srcName := escapeDriveColon(JoinPathes(srcPath))
		objName := JoinPathes(objdir, dstPathBase+".o")
		depName := JoinPathes(objdir, dstPathBase+".d")
		createList = append(createList, objName)

		carg := make([]string, 0, len(arg1)+len(info.options))
		carg = append(carg, arg1...)
		for _, ca := range info.options {
			switch ca {
			case "$out":
				ca = objName
			case "$dep":
				ca = depName
			case "$in":
				ca = srcName
			default:
				/* NO-OP */
			}
			carg = append(carg, ca)
		}
		srcExt := filepath.Ext(srcPath)
		if rule, exists := otherRuleList[srcExt]; exists {
			if compiler, ok := info.variables[rule.Compiler]; ok {
				compiler, err := info.Interpolate(compiler)
				if err != nil {
					return []string{}, err
				}

				ocmd := OtherRuleFile{
					Rule:     "compile" + srcExt,
					Compiler: compiler,
					Infile:   srcName,
					Outfile:  objName,
					Include: (func() string {
						if !rule.needInclude {
							return ""
						}
						return strings.Join(info.includes, " ")
					})(),
					Option: (func() string {
						opts := make([]string, 0, len(rule.Options))
						for _, o := range rule.Options {
							switch o {
							case "$out":
								opts = append(opts, objName)
							case "$in":
								opts = append(opts, srcName)
							case "$dep":
								opts = append(opts, depName)
							default:
								opts = append(opts, o)
							}
						}
						return strings.Join(opts, " ")
					})(),
					Define: (func() string {
						if !rule.needDefine {
							return ""
						}
						return strings.Join(info.defines, " ")
					})()}
				if rule.NeedDepend == true {
					ocmd.Depend = depName
				}
				otherRuleFileList = append(otherRuleFileList, ocmd) // Record it
			} else {
				Warn("compiler: Missing a compiler \"%s\" definitions in \"%s\".",
					rule.Compiler,
					JoinPathes(info.mydir, "make.yml"))
			}
		} else {
			// normal
			cmd := BuildCommand{
				Command:          compiler,
				CommandType:      "compile",
				Args:             carg,
				InFiles:          []string{srcName},
				OutFile:          objName,
				DepFile:          depName,
				NeedCommandAlias: true}
			if 0 < len(pchFile) {
				cmd.ImplicitDepends = append(cmd.ImplicitDepends, pchFile)
				cmd.Args = append(cmd.Args, "-include-pch", pchFile)
			}
			commandList = append(commandList, cmd) // Record it
		}
	}
	return createList, nil
}

// Create pre-compiled header if possible.
func createPCH(info BuildInfo, srcdir string, compiler string) string {
	const pchName = "00-common-prefix.hpp"
	pchSrc := JoinPathes(srcdir, pchName)
	if !Exists(pchSrc) {
		Verbose("%s: \"%s\" is not detected.\n", ProgramName, pchSrc)
		return ""
	}
	Verbose("%s: \"%s\" found.\n", ProgramName, pchSrc)
	pchDst := JoinPathes(info.outputdir, srcdir, buildDirectory, pchName+".pch")
	Verbose("%s: Create PCH \"%s\"\n", ProgramName, pchDst)
	args := append(info.includes, info.defines...)
	for _, opt := range info.options {
		args = append(args, (func(o string) string {
			switch o {
			case "$out":
				return pchDst
			case "$dep":
				return pchDst + ".dep"
			case "$in":
				return pchSrc
			default:
				return o
			}
		})(opt))
	}
	// PCH source found.
	Verbose("%s: PCH creation command line is \"%s\".\n", ProgramName, strings.Join(args, " "))
	cmd := BuildCommand{
		Command:          compiler,
		CommandType:      "gen_pch",
		Args:             args,
		InFiles:          []string{pchSrc},
		OutFile:          pchDst,
		DepFile:          pchDst + ".dep",
		NeedCommandAlias: true}
	commandList = append(commandList, cmd)
	return pchDst
}

// Exists checks `filename` existence.
func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

//
// other rule
//
func createOtherRule(info BuildInfo, olist []Other, optionPrefix string) error {
	for _, ot := range olist {
		if ot.Type != "" && ot.Type != option.targetType {
			continue
		}

		ext := ot.Ext

		olist := []string{}
		for _, o := range getList(ot.Option, info.target) {
			var e error
			olist, e = appendOption(info, olist, o, optionPrefix)
			if e != nil {
				return e
			}
		}

		needInclude := false
		needOption := false
		needDefine := false
		rule, ok := otherRuleList[ext]
		if ok {
			rule.Options = append(rule.Options, olist...)
		} else {
			// no exist rule
			cmdl := strings.Split(ot.Command, " ")
			compiler := ""

			cmdline := "$compiler"
			for i, c := range cmdl {
				if i == 0 {
					compiler = c
				} else if c[0] == '@' {
					switch c {
					case "@include":
						needInclude = true
					case "@option":
						needOption = true
					case "@define":
						needDefine = true
					}
					cmdline += " $" + c[1:]
				} else {
					cmdline += " " + c
				}
			}

			rule = OtherRule{
				Compiler:    compiler,
				Command:     cmdline,
				Title:       ot.Description,
				Options:     olist,
				needInclude: needInclude,
				needOption:  needOption,
				needDefine:  needDefine,
				NeedDepend:  ot.NeedDepend}
		}
		otherRuleList[ext] = rule
	}
	return nil
}

func checkType(vlist []Variable) string {
	for _, v := range vlist {
		if v.Name == "default_type" {
			return v.Value
		}
	}
	return "default"
}

// Creates *.ninja file.
func outputNinja() error {
	Verbose("%s: Creates \"%s\"\n", ProgramName, option.ninjaFile)

	tPath := NewTransientOutput(option.ninjaFile)
	file, err := os.Create(tPath.TempOutput)
	if err != nil {
		return errors.Wrapf(err, "failed to create temporal output \"%s\"", tPath.TempOutput)
	}
	defer file.Close()
	defer tPath.Abort()
	Verbose("%s: Creating transient output \"%s\"\n", ProgramName, tPath.TempOutput)
	sink := bufio.NewWriter(file)
	// execute build
	if err = outputRules(sink); err != nil {
		return errors.Wrapf(err, "failed to emit rules.")
	}

	// Emits rules for updating `build.ninja`
	type WriteContext struct {
		Commands      []BuildCommand
		OtherRules    []OtherRuleFile
		SubNinjas     []string
		NinjaFile     string
		ConfigSources []string
	}
	ctx := WriteContext{
		Commands:      commandList,
		OtherRules:    otherRuleFileList,
		SubNinjas:     subNinjaList,
		NinjaFile:     option.ninjaFile,
		ConfigSources: ScannedConfigs}
	funcs := template.FuncMap{
		"escape_drive": escapeDriveColon,
		"join":         strings.Join}
	commandTemplate := template.Must(template.New("rules").Funcs(funcs).Parse(`# Commands
{{- define "IMPDEPS"}}
    {{- if .}} | {{join . " "}}{{end}}
{{- end}}
build {{.NinjaFile}} : update_ninja_file {{join .ConfigSources " "}}
    desc = {{.NinjaFile}}
{{range $c := .Commands}}
build {{$c.OutFile}} : {{$c.CommandType}} {{join $c.InFiles " "}} {{join $c.Depends " "}} {{template "IMPDEPS" $c.ImplicitDepends}}
    desc = {{$c.OutFile}}
{{- if $c.NeedCommandAlias}}
    {{$c.CommandType}} = {{$c.Command}}
{{- end}}
{{- if $c.DepFile}}
    depf = {{$c.DepFile}}
{{- end}}
{{- if $c.Args}}
    options = {{join $c.Args " "}}
{{- end}}
{{end}}
# Other rules
{{range $item := .OtherRules}}
build {{$item.Outfile}} : {{$item.Rule}} {{$item.Infile}}
    desc     = {{$item.Outfile}}
    compiler = {{$item.Compiler}}
{{- if $item.Include}}
    include  = {{$item.Include}}
{{- end}}
{{- if  $item.Option}}
    option   = {{$item.Option}}
{{- end}}
{{- if  $item.Depend}}
    depf     = {{$item.Depend}}
{{- end}}
{{end}}
{{- if .SubNinjas}}
{{range $subninja := .SubNinjas}}
subninja {{$subninja}}
{{end}}
{{end}}
`))
	commandTemplate.Execute(sink, ctx)

	sink.Flush()

	if err := file.Close(); err != nil {
		return errors.Wrapf(err, "closing \"%s\" failed.", file.Name())
	}
	if err := tPath.Commit(); err != nil {
		return errors.Wrapf(err, "renaming \"%s\" to \"%s\" failed.", tPath.TempOutput, tPath.Output)
	}
	Verbose("%s: Renaming %s to %s\n", ProgramName, tPath.TempOutput, tPath.Output)
	return nil
}

//// Construct a properly folded string from `args`.
//func fold(args []string, maxColumns int, prefix string) string {
//	lines := make([]string, 0, 8)
//	line := ""
//	maxcol := maxColumns - len(prefix)
//	emptyPrefix := strings.Repeat(" ", len(prefix))
//	for _, arg := range args {
//		if maxcol < len(line)+1+len(arg) {
//			lines = append(lines, prefix+line)
//			line = ""
//			prefix = emptyPrefix
//		}
//		line += " " + arg
//	}
//	if 0 < len(line) {
//		lines = append(lines, prefix+line)
//	}
//	return strings.Join(lines, " $\n")
//}

// Emits common rules.
func outputRules(file io.Writer) error {
	type RuleContext struct {
		Platform           string
		UseResponse        bool
		NewlineAsDelimiter bool
		GroupArchives      bool
		OutputDirectory    string
		OtherRules         map[string]OtherRule
		AppendRules        map[string]AppendBuild
		UsePCH             bool
		NinjaUpdater       string
	}
	//println("Platform: " + targetType)
	tmpl := template.Must(template.New("common").Parse(`# Rule definitions
builddir = {{.OutputDirectory}}

rule compile
    description = Compiling: $desc
{{- if eq .Platform "WIN32"}}
    command = $compile $options -Fo$out $in
    deps = msvc
{{- else}}
    command = $compile $options -o $out $in
    depfile = $depf
    deps = gcc
{{- end}}

{{- if .UsePCH}}
rule gen_pch
    description = Create PCH: $desc
    command = $gen_pch $options -x c++-header -o $out $in
    depfile = $depf
    deps = gcc
{{- end}}

rule ar
    description = Archiving: $desc
{{- if .UseResponse}}
    command = $ar $options {{if eq .Platform "WIN32"}}/out:$out{{else}}$out{{end}} @$out.rsp
    rspfile = $out.rsp
    rspfile_content = {{if .NewlineAsDelimiter}}$in_newline{{else}}$in{{end}}
{{- else}}
    command = $ar $options $out $in
{{- end}}

rule link
    description = Linking: $desc
{{- if .UseResponse}}
    {{- if eq .Platform "WIN32"}}
    command = $link $options /out:$out @$out.rsp
    {{- else}}
    command = $link $options -o $out @$out.rsp
    {{- end}}
    rspfile = $out.rsp
    rspfile_content = {{if .NewlineAsDelimiter}}$in_newline{{else}}$in{{end}}
{{- else}}
    {{- if .GroupArchives}}
    command = $link $options -o $out -Wl,--start-group $in -Wl,--end-group
    {{- else}}
    command = $link -o $out $in $options
    {{- end}}
{{- end}}

rule packager
    description = Packaging: $desc
    command = $packager $options $in $out

rule convert
    description = Converting: $desc
    command = $convert $options -o $out $in
{{range $k, $v := .OtherRules}}
rule compile{{- $k}}
    description = {{$v.Title}}: $desc
    command = {{$v.Command}}
    {{- if $v.NeedDepend}}
    depfile = $depf
    deps = gcc
    {{- end}}
{{end}}
{{range $k, $v := .AppendRules}}
rule {{$k}}
    description = {{$v.Desc}}: $desc
    command = {{$v.Command}}
    {{- if $v.Deps}}
    depfile = $out.d
    deps = gcc
    {{- end}}
{{end}}
rule update_ninja_file
    description = Update $desc
    command     = {{.NinjaUpdater}}
    generator   = 1

build always: phony
# end of [Rule definitions]
`))

	ctx := RuleContext{
		Platform:           option.targetType,
		UseResponse:        useResponse,
		NewlineAsDelimiter: responseNewline,
		GroupArchives:      groupArchives,
		OutputDirectory:    filepath.ToSlash(option.outputDir),
		OtherRules:         otherRuleList,
		AppendRules:        appendRules,
		UsePCH:             true,
		NinjaUpdater:       strings.Join(os.Args, " ")}

	return tmpl.Execute(file, ctx)
}

// Creates *.vcxproj (for VisualStudio).
func outputMSBuild(outdir, projname string) error {
	var targets []string

	for _, command := range commandList {
		if command.CommandType != "compile" {
			continue
		}

		for _, infile := range command.InFiles {
			targets = append(targets, unescapeDriveColon(infile))
		}
	}

	msbuild.ExportProject(targets, outdir, projname)
	return nil
}

// JoinPathes joins suppiled path components and normalize the result.
func JoinPathes(pathes ...string) string {
	return filepath.ToSlash(filepath.Clean(filepath.Join(pathes...)))
}

// Escapes ':' in path.
func escapeDriveColon(path string) string {
	if filepath.IsAbs(path) && strings.Index(path, ":") == 1 {
		drive := filepath.VolumeName(path)
		if 0 < len(drive) {
			drive = strings.Replace(strings.ToLower(drive), ":", "$:", 1)
			path = drive + path[2:]
		}
	}
	return path
}

// Convert back to escaped path
func unescapeDriveColon(path string) string {
	return strings.Replace(path, "$:", ":", 1)
}

// Verbose output if wanted
func Verbose(format string, args ...interface{}) {
	if option.verbose {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

// Warn emit a warning to `os.Stderr`
func Warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s:warning:", ProgramName)
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
}

// ToBoolean converts passed string to boolean.
func ToBoolean(s string) bool {
	if rxTruthy.MatchString(s) {
		return true
	}
	if rxFalsy.MatchString(s) {
		return false
	}
	Warn("Ambiguous boolean \"%s\" found", s)
	return false
}

func (v *Variable) getValue(info *BuildInfo) (result string, ok bool) {
	if v.Type != "" && v.Type != option.targetType {
		return
	}
	if v.Target != "" && v.Target != info.target {
		return
	}
	if v.Build != "" {
		bld := strings.ToLower(v.Build)
		switch option.variant {
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
