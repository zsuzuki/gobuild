//
//
//
package main

import (
	"flag"
	"fmt"
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
	isDebug           bool
	isRelease         bool
	isProduct         bool
	isDevelop         bool
	isDevelopRelease  bool
	targetType        string
	targetName        string
	outputRootDir     string
	appendRules       map[string]AppendBuild
	otherRuleList     map[string]OtherRule
	commandList       []BuildCommand
	otherRuleFileList []OtherRuleFile
	verboseMode       bool
	useResponse       bool
	groupArchives     bool
	responseNewline   bool
	buildNinjaName    string
	subNinjaList      []string
	ProgramName       = getExeName()

	rx_truthy = regexp.MustCompile(`^\s*(?i:t(?:rue)?|y(?:es)?|on|1)(?:\s+.*)?$`)
	rx_falsy  = regexp.MustCompile(`^\s*(?i:f(?:alse)|no?|off|0)(?:\s+.*)?$`)
)

// The entry point.
func main() {

	//ProgramName := getExeName()

	gen_msbuild := false
	projdir := ""
	projname := ""

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [<target>]\n", ProgramName)
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.BoolVar(&verboseMode, "v", false, "verbose mode")
	flag.BoolVar(&isRelease, "release", false, "release build")
	flag.BoolVar(&isDebug, "debug", true, "debug build")
	flag.BoolVar(&isDevelop, "develop", false, "develop(beta) build")
	flag.BoolVar(&isDevelopRelease, "develop_release", false, "develop(beta) release build")
	flag.BoolVar(&isProduct, "product", false, "for production build")
	flag.StringVar(&targetType, "type", "default", "build target type")
	flag.StringVar(&targetName, "t", "", "build target name")
	flag.StringVar(&outputRootDir, "o", "build", "build directory")
	flag.StringVar(&buildNinjaName, "f", "build.ninja", "output build.ninja filename")
	flag.BoolVar(&gen_msbuild, "msbuild", false, "Export MSBuild project")
	flag.StringVar(&projdir, "msbuild-dir", "./", "MSBuild project output directory")
	flag.StringVar(&projname, "msbuild-proj", "out", "MSBuild project name")
	showVersionAndExit := flag.Bool("version", false, "display version")
	flag.Parse()

	if *showVersionAndExit {
		fmt.Fprintf(os.Stdout, "%s: %v (%s/%s)\n", ProgramName, cbuildVersion, runtime.Version(), runtime.Compiler)
		os.Exit(0)
	}

	switch {
	case isDevelop:
		isDebug = false
	case isDevelopRelease:
		isDevelop = false
		isDebug = false
	case isRelease:
		isDevelopRelease = false
		isDevelop = false
		isDebug = false
	case isProduct:
		isDevelopRelease = false
		isRelease = false
		isDevelop = false
		isDebug = false
	}
	useResponse = false
	groupArchives = false
	responseNewline = false

	ra := flag.Args()
	if 0 < flag.NArg() && targetName == "" {
		targetName = ra[0]
	}

	appendRules = map[string]AppendBuild{}
	commandList = []BuildCommand{}
	otherRuleList = map[string]OtherRule{}
	otherRuleFileList = []OtherRuleFile{}
	subNinjaList = []string{}

	if targetName != "" {
		Verbose("%s: Target is \"%s\"\n", ProgramName, targetName)
	}
	buildinfo := BuildInfo{
		variables:      map[string]string{"option_prefix": "-"},
		includes:       []string{},
		defines:        []string{},
		options:        []string{},
		archiveOptions: []string{},
		convertOptions: []string{},
		linkOptions:    []string{},
		selectTarget:   targetName,
		target:         targetName}

	if _, err := CollectConfigurations(buildinfo, ""); err != nil {
		fmt.Fprintf(os.Stderr, "%s:error: %v\n", ProgramName, err)
		os.Exit(1)
	}
	if nlen := len(commandList) + len(otherRuleFileList); nlen <= 0 {
		fmt.Fprintf(os.Stderr, "%s: No commands to run.\n", ProgramName)
		os.Exit(0)
	}
	if gen_msbuild {
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

// Collects configurations recursively.
func CollectConfigurations(info BuildInfo, outputDirectory string) (*[]string, error) {
	var odir string
	if outputDirectory == "" {
		odir = "./"
	} else {
		odir = filepath.ToSlash(filepath.Clean(outputDirectory)) + "/"
	}
	return traverse(info, odir, 0)
}

func traverse(info BuildInfo, outputDir string, level int) (*[]string, error) {
	// result.Successed = false // Named return value is initialized to 0
	if outputDir[len(outputDir)-1] != '/' {
		return nil, errors.New("Output directory should end with '/'")
	}
	Verbose("%s: Enter \"%s\"\n", ProgramName, outputDir)
	defer Verbose("%s: Leave \"%s\"\n", ProgramName, outputDir)

	yamlSource := filepath.Join(outputDir, "make.yml")
	buf, err := ioutil.ReadFile(yamlSource)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read \"%s\"", yamlSource)
	}

	var d Data
	err = yaml.Unmarshal(buf, &d)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to unmarshal \"%s\"", yamlSource)
	}

	info.mydir = outputDir
	//
	// select target
	//
	currentTarget, _, ok := getTarget(info, d.Target)
	if !ok {
		return nil, errors.New("No targets.")
	}
	if info.target == "" {
		info.target = currentTarget.Name
		Verbose("%s: Target is \"%s\".\n", ProgramName, info.target)
	}
	info.selectTarget = ""

	if level == 0 && targetType == "default" {
		targetType = checkType(d.Variable)
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
		switch {
		case isProduct:
			outputRootDir = JoinPathes(outputRootDir, targetType, "Product")
		case isDevelop:
			outputRootDir = JoinPathes(outputRootDir, targetType, "Develop")
		case isDevelopRelease:
			outputRootDir = JoinPathes(outputRootDir, targetType, "DevelopRelease")
		case isRelease:
			outputRootDir = JoinPathes(outputRootDir, targetType, "Release")
		default:
			outputRootDir = JoinPathes(outputRootDir, targetType, "Debug")
		}
	}

	info.outputdir = JoinPathes(outputRootDir, outputDir) + "/" // Proofs '/' ending

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
				info.AddInclude(JoinPathes(outputDir, pth))
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
	cvfiles := getList(d.Convert_List, info.target)
	testfiles := getList(d.Tests, info.target)

	// sub-directories
	subdirs := getList(d.Subdir, info.target)

	subArtifacts := []string{}

	// Recurse into the sub-directories.
	for _, s := range subdirs {
		// outputDir always ends with '/'
		odir := outputDir + filepath.ToSlash(filepath.Clean(s)) + "/"
		if r, err := traverse(info, odir, level+1); err == nil {
			if 0 < len(*r) {
				subArtifacts = append(subArtifacts, *r...)
			}
		} else {
			return nil, err
		}
	}

	// pre build files
	if err = createPrebuild(info, outputDir, d.Prebuild); err != nil {
		return nil, err
	}

	// create compile list
	artifacts, err := compileFiles(info, outputDir, files)
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
			Warn("There are no files to build in \"%s\".", outputDir)
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
			Warn("There are no files to build in \"%s\".", outputDir)
		}
	case "convert":
		if 0 < len(cvfiles) {
			createConvert(info, outputDir, cvfiles, currentTarget.Name)
		} else {
			Warn("There are no files to convert in \"%s\".", outputDir)
		}
	case "passthrough":
		// Just bubbling up the artifacts
		result = append(subArtifacts, artifacts...)
	case "test":
		// unit tests
		if e := createTest(info, testfiles, outputDir); e != nil {
			return nil, e
		}
	default:
		/* NO-OP */
	}

	Verbose("%s: Artifacts in \"%s\":\n", ProgramName, outputDir)
	if verboseMode {
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
		if item.Match(targetName, targetType) {
			lists = append(lists, item.List...)
			switch {
			case isDebug:
				lists = append(lists, item.Debug...)
			case isDevelop:
				lists = append(lists, item.Develop...)
			case isRelease:
				lists = append(lists, item.Release...)
			case isDevelopRelease:
				lists = append(lists, item.DevelopRelease...)
			case isProduct:
				lists = append(lists, item.Product...)
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

	var archiveName string
	switch targetType {
	case "WIN32":
		archiveName = targetName + ".lib"
	default:
		archiveName = fmt.Sprintf("lib%s.a", targetName)
	}
	archiveName = JoinPathes(info.outputdir, archiveName)

	arCommand, e := info.ExpandVariable("archiver")
	if e != nil {
		return "", e
	}

	cmd := BuildCommand{
		Command:          arCommand,
		CommandType:      "ar",
		Args:             info.archiveOptions,
		InFiles:          inputs,
		OutFile:          archiveName,
		NeedCommandAlias: true}
	commandList = append(commandList, cmd) // Record command

	return archiveName, nil
}

//
// link objects
//
func createLink(info BuildInfo, createList []string, targetName string, packager Packager) error {
	trname := targetName
	esuf, ok := info.variables["execute_suffix"]
	if ok {
		trname += esuf
	}
	trname = JoinPathes(info.outputdir, trname)

	linker, e := info.ExpandVariable("linker")
	if e != nil {
		return e
	}

	options := []string{}
	for _, lo := range info.linkOptions {
		lo = strings.Replace(lo, "$out", trname, -1)
		options = append(options, lo)
	}
	options = append(options, info.libraries...)

	// execute
	cmd := BuildCommand{
		Command:          linker,
		CommandType:      "link",
		Args:             options,
		InFiles:          createList,
		OutFile:          trname,
		Depends:          info.linkDepends,
		NeedCommandAlias: true}
	commandList = append(commandList, cmd)
	//fmt.Println("-o " + NowTarget.Name + flist)

	if packager.Target != "" {
		// package
		pkgname := JoinPathes(outputRootDir, targetName, packager.Target)
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
			InFiles:          []string{trname},
			OutFile:          pkgname,
			NeedCommandAlias: true}
		commandList = append(commandList, pkg) // Record commands
	}
	return nil
}

//
// convert objects
//
func createConvert(info BuildInfo, loaddir string, createList []string, targetName string) {
	cvname := JoinPathes(info.outputdir, targetName)
	converter := info.variables["converter"]

	clist := []string{}
	for _, f := range createList {
		clist = append(clist, JoinPathes(loaddir+f))
	}

	cmd := BuildCommand{
		Command:          converter,
		CommandType:      "convert",
		Args:             info.convertOptions,
		InFiles:          clist,
		OutFile:          cvname,
		NeedCommandAlias: true}
	commandList = append(commandList, cmd) // Record commands
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
		if len(tlist) > 0 {
			t := tlist[0]
			if info.target == "" {
				return t, "_" + t.Name, true
			}
			return t, "", true
		}
	}
	return Target{}, "", false
}

//
func replacePath(value string, addDir string) (string, string) {
	url := strings.Split(value, " ")
	ucmd := url[0]
	if ucmd[0] == '$' {
		ucmd = ucmd[1:]
	}
	p := JoinPathes(addDir, ucmd)
	result := p
	for i, uu := range url {
		if i > 0 {
			result += " " + uu
		}
	}
	return result, p
}

//
// compile files
//
func createPrebuild(info BuildInfo, loaddir string, plist []Build) error {
	for _, p := range plist {
		if !p.Match(info.target, targetType) {
			continue
		}

		// register prebuild
		sources := getList(p.Source, info.target)
		if len(sources) == 0 {
			return fmt.Errorf("No sources for command `%s`", p.Name)
		}
		for i, src := range sources {
			if src[0] == '$' {
				sabs, _ := filepath.Abs(filepath.Join(info.outputdir, "output", src[1:]))
				sources[i] = escapeDriveColon(JoinPathes(sabs))
			} else if src == "always" {
				sources[i] = "always |"
			} else {
				if expanded, err := info.Interpolate(src); err == nil {
					sources[i] = JoinPathes(loaddir, expanded)
				} else {
					return err
				}
			}
		}
		ur, ok := info.variables[p.Command]
		if !ok {
			return errors.Errorf("Missing build command \"%s\" (referenced from \"%s\")", p.Command, p.Name)
		}
		mycmd := strings.Replace(JoinPathes(info.outputdir, p.Command), "/", "_", -1)
		deps := []string{}

		if _, af := appendRules[mycmd]; !af {
			ur = strings.Replace(ur, "${selfdir}", loaddir, -1)
			var err error
			ur, err = info.Interpolate(ur)
			if err != nil {
				return err
			}
			if ur[0] == '$' {
				r, d := replacePath(ur, info.outputdir)
				abs, _ := filepath.Abs(d)
				d = filepath.ToSlash(abs)
				deps = append(deps, d)
				ur = r
			} else if strings.HasPrefix(ur, "../") || strings.HasPrefix(ur, "./") {
				r, d := replacePath(ur, loaddir)
				deps = append(deps, d)
				ur = r
			}
			ur = strings.Replace(ur, "$target", info.target, -1)
			useDeps := false
			if p.Deps != "" {
				useDeps = true
			}
			ab := AppendBuild{
				Command: ur,
				Desc:    p.Command,
				Deps:    useDeps}
			appendRules[mycmd] = ab
		}

		if p.Name[0] != '$' || strings.HasPrefix(p.Name, "$target/") {
			pn := p.Name
			if pn[0] == '$' {
				pn = strings.Replace(pn, "$target/", "/."+info.target+"/", 1)
			}
			outfile, _ := filepath.Abs(filepath.Join(info.outputdir, pn))
			outfile = escapeDriveColon(JoinPathes(outfile))
			cmd := BuildCommand{
				Command:          p.Command,
				CommandType:      mycmd,
				Depends:          deps,
				InFiles:          sources,
				OutFile:          outfile,
				NeedCommandAlias: false}
			commandList = append(commandList, cmd)
		} else {
			// Found `$...`
			ext := p.Name[1:] //filepath.Ext(p.Name)
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
					Command:          p.Command,
					CommandType:      mycmd,
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
				dstPathBase = strings.Replace(dstPathBase, "$target/", "/."+info.target+"/", 1)
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
			// custom
			linc := ""
			if rule.needInclude {
				linc = strings.Join(info.includes, " ")
			}
			ldef := ""
			if rule.needDefine {
				ldef = strings.Join(info.defines, " ")
			}
			lopt := ""
			{
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
				lopt = strings.Join(opts, " ")
			}
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
					Include:  linc,
					Option:   lopt,
					Define:   ldef,
					Depend:   ""}
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
	pchSrc := filepath.ToSlash(filepath.Join(srcdir, pchName))
	if !Exists(pchSrc) {
		Verbose("%s: \"%s\" does not exists.\n", ProgramName, pchSrc)
		return ""
	}
	Verbose("%s: \"%s\" found.\n", ProgramName, pchSrc)
	dstdir := filepath.Join(info.outputdir, srcdir, buildDirectory)
	pchDst := filepath.ToSlash(filepath.Join(dstdir, pchName+".pch"))
	Verbose("%s: Create PCH \"%s\"\n", ProgramName, pchDst)
	args := append(info.includes, info.defines...)
	for _, opt := range info.options {
		switch opt {
		case "$out":
			args = append(args, pchDst)
		case "$dep":
			args = append(args, pchDst+".dep")
		case "$in":
			args = append(args, pchSrc)
		default:
			args = append(args, opt)
		}
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

func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

//
// other rule
//
func createOtherRule(info BuildInfo, olist []Other, optionPrefix string) error {
	for _, ot := range olist {
		if ot.Type != "" && ot.Type != targetType {
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
	Verbose("%s: Creates \"%s\"\n", ProgramName, buildNinjaName)

	tPath := NewTransientOutput(buildNinjaName)
	file, err := os.Create(tPath.TempOutput)
	if err != nil {
		return errors.Wrapf(err, "Failed to create temporal output \"%s\"", tPath.TempOutput)
	}
	defer file.Close()
	defer tPath.Abort()
	Verbose("%s: Creating transient output \"%s\"\n", ProgramName, tPath.TempOutput)

	// execute build
	if err = outputRules(file); err != nil {
		return errors.Wrapf(err, "Failed to emit rules.")
	}
	type WriteContext struct {
		Commands   []BuildCommand
		OtherRules []OtherRuleFile
		SubNinjas  []string
	}
	ctx := WriteContext{
		Commands:   commandList,
		OtherRules: otherRuleFileList,
		SubNinjas:  subNinjaList}
	funcs := template.FuncMap{"escape_drive": escapeDriveColon}
	commandTemplate := template.Must(template.New("rules").Funcs(funcs).Parse(`# Commands
{{- define "INFILES"}}
    {{- range $in := .}} {{$in}}{{end}}
{{- end}}
{{- define "DEPS"}}
    {{- if .}}{{range $dep := .}} {{$dep}}{{end}}{{end}}
{{- end}}
{{- define "IMPDEPS"}}
    {{- if .}} |{{range $impdep := .}} {{$impdep}}{{end}}{{- end}}
{{- end}}
{{range $c := .Commands}}
build {{$c.OutFile}} : {{$c.CommandType}}{{template "INFILES" $c.InFiles}}{{template "DEPS" $c.Depends}}{{template "IMPDEPS" $c.ImplicitDepends}}
    desc = {{$c.OutFile}}
{{- if $c.NeedCommandAlias}}
    {{$c.CommandType}} = {{$c.Command}}
{{- end}}
{{- if $c.DepFile}}
    depf = {{$c.DepFile}}
{{- end}}
{{- if $c.Args}}
    options ={{range $a := $c.Args}} {{$a}}{{end}}
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
{{- if $item.Option}}
    option   = {{$item.Option}}
{{- end}}
{{- if $item.Depend}}
    depf     = {{$item.Depend}}
{{- end}}
{{- end}}
{{- if .SubNinjas}}
{{range $subninja := .SubNinjas}}
subninja {{$subninja}}
{{end}}
{{end}}
`))
	commandTemplate.Execute(file, ctx)
	if err := file.Close(); err != nil {
		return errors.Wrapf(err, "Closing \"%s\" failed.", file.Name())
	}
	if err := tPath.Commit(); err != nil {
		return errors.Wrapf(err, "Renaming \"%s\" to \"%s\" failed.", tPath.TempOutput, tPath.Output)
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
func outputRules(file *os.File) error {
	type RuleContext struct {
		Platform           string
		UseResponse        bool
		NewlineAsDelimiter bool
		GroupArchives      bool
		OutputDirectory    string
		OtherRules         map[string]OtherRule
		AppendRules        map[string]AppendBuild
		UsePCH             bool
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
    {{- if eq .Platform "WIN32"}}
    command = $ar $options /out:$out @$out.rsp
    {{- else}}
    command = $ar $options $out @$out.rsp
    {{- end}}
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
build always: phony
# end of [Rule definitions]
`))

	ctx := RuleContext{
		Platform:           targetType,
		UseResponse:        useResponse,
		NewlineAsDelimiter: responseNewline,
		GroupArchives:      groupArchives,
		OutputDirectory:    filepath.ToSlash(outputRootDir),
		OtherRules:         otherRuleList,
		AppendRules:        appendRules,
		UsePCH:             true}

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

// Joins suppiled path components and normalize the result.
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
	if verboseMode {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

// Emit a warning to `os.Stderr`
func Warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s:warning:", ProgramName)
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
}

// Converts passed string to boolean.
func ToBoolean(s string) bool {
	if rx_truthy.MatchString(s) {
		return true
	}
	if rx_falsy.MatchString(s) {
		return false
	}
	Warn("Ambiguous boolean \"%s\" found", s)
	return false
}

func (v *Variable) getValue(info *BuildInfo) (result string, ok bool) {
	if v.Type != "" && v.Type != targetType {
		return
	}
	if v.Target != "" && v.Target != info.target {
		return
	}
	if v.Build != "" {
		bld := strings.ToLower(v.Build)
		if isDebug && bld != "debug" {
			return
		}
		if isRelease && bld != "release" {
			return
		}
		if isDevelop && bld != "develop" {
			return
		}
		if isDevelopRelease && bld != "develop_release" {
			return
		}
		if isProduct && bld != "product" {
			return
		}
	}
	return v.Value, true
}
