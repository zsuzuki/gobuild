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
	"runtime"
	"strings"
	"text/template"

	"github.com/kuma777/go-msbuild"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
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
	toplevel          bool
	outputdir         string
	outputdirSet      bool
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
	ProgramName       string
)

// The entry point.
func main() {

	ProgramName := getExeName()

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
	flag.StringVar(&outputdir, "o", "build", "build directory")
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
	outputdirSet = false
	useResponse = false
	groupArchives = false
	toplevel = true
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
	if r, err := build(buildinfo, ""); !r.success {
		fmt.Fprintf(os.Stderr, "%s:error: %v\n", ProgramName, err)
		os.Exit(1)
	}
	if nlen := len(commandList) + len(otherRuleFileList); nlen <= 0 {
		fmt.Fprintf(os.Stderr, "%s: No commands to run.\n", ProgramName)
		os.Exit(0)
	}

	Verbose("%s: Creates \"%s\".\n", ProgramName, buildNinjaName)
	if err := outputNinja(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v", ProgramName, err)
		os.Exit(1)
	}
	if gen_msbuild {
		Verbose("%s: Creates VC++ project file(s).\n", ProgramName)
		outputMSBuild(projdir, projname)
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

//
// writing rules
//
func build(info BuildInfo, pathname string) (result BuildResult, err error) {
	loaddir := pathname
	if loaddir == "" {
		loaddir = "./"
	} else {
		loaddir += "/"
	}
	Verbose("%s: Enter %s\n", pathname)
	myYaml := filepath.Join(loaddir, "make.yml")
	buf, err := ioutil.ReadFile(myYaml)
	if err != nil {
		result.success = false
		return result, errors.Wrapf(err, "Failed to read \"%s\"", myYaml)
	}

	var d Data
	err = yaml.Unmarshal(buf, &d)
	if err != nil {
		result.success = false
		return result, errors.Wrapf(err, "Failed to unmarshal \"%s\"", myYaml)
	}

	info.mydir = loaddir
	//
	// select target
	//
	NowTarget, _, ok := getTarget(info, d.Target)
	if !ok {
		result.success = false
		return result, errors.New("No targets.")
	}
	if info.target == "" {
		info.target = NowTarget.Name
		Verbose("%s: Target is \"%s\".\n", ProgramName, info.target)
	}
	info.selectTarget = ""

	if toplevel == true && targetType == "default" {
		targetType = checkType(d.Variable)
	}
	toplevel = false
	//
	// get rules
	//
	newvar := map[string]string{}
	for ik, iv := range info.variables {
		newvar[ik] = iv
	}
	info.variables = newvar
	for _, v := range d.Variable {
		if val, ok := getVariable(info, v); ok {
			switch v.Name {
			case "enable_response":
				switch val {
				case "true":
					useResponse = true
				case "false":
					useResponse = false
				default:
					Warning("Unrecognized value [%s] for `enable_response`\n", v.Value)
				}
			case "response_newline":
				switch val {
				case "true":
					responseNewline = true
				case "false":
					responseNewline = false
				default:
					Warning("Unrecognized value [%s] for `response_newline`\n", v.Value)
				}
			case "group_archives":
				switch val {
				case "true":
					groupArchives = true
				case "false":
					groupArchives = false
				default:
					Warning("Unrecognized value [%s] for `group_archives`\n", v.Value)
				}
			default: /* NO-OP */
			}
			info.variables[v.Name] = val
		}
	}
	optionPrefix := info.OptionPrefix()
	if outputdirSet == false {
		switch {
		case isProduct:
			outputdir = filepath.Join(outputdir, targetType, "Product")
		case isDevelop:
			outputdir = filepath.Join(outputdir, targetType, "Develop")
		case isDevelopRelease:
			outputdir = filepath.Join(outputdir, targetType, "DevelopRelease")
		case isRelease:
			outputdir = filepath.Join(outputdir, targetType, "Release")
		default:
			outputdir = filepath.Join(outputdir, targetType, "Debug")
		}
		outputdirSet = true
	}

	info.outputdir = normalizePath(filepath.Join(outputdir, loaddir)) + "/" // Proofs '/' ending

	//objdir := filepath.Clean(filepath.Join(outputdir, loaddir, buildDirectory+objsSuffix)) + "/" // Proofs '/' ending

	for _, pth := range getList(d.Include, info.target) {
		if strings.HasPrefix(pth, "$output") {
			pth = filepath.Clean(filepath.Join(info.outputdir, "output"+pth[7:]))
		} else {
			useRel := pth[0] == '$'
			var err error
			pth, err = info.StrictInterpolate(pth)
			if err != nil {
				result.success = false
				return result, err
			}
			if !useRel && !filepath.IsAbs(pth) {
				pth = filepath.Clean(filepath.Join(loaddir, pth))
			}
		}
		info.AddInclude(pth)
	}
	for _, d := range getList(d.Define, info.target) {
		info.AddDefines(d)
		//info.defines = append(info.defines, optionPrefix+"D"+d)
	}
	for _, o := range getList(d.Option, info.target) {
		info.options, err = appendOption(info, info.options, o, optionPrefix)
		if err != nil {
			result.success = false
			return result, err
		}
	}
	for _, a := range getList(d.ArchiveOption, info.target) {
		info.archiveOptions, err = appendOption(info, info.archiveOptions, a, "")
		if err != nil {
			result.success = false
			return result, err
		}
	}
	for _, c := range getList(d.ConvertOption, info.target) {
		info.convertOptions, err = appendOption(info, info.convertOptions, c, "")
		if err != nil {
			result.success = false
			return result, err
		}
	}
	for _, l := range getList(d.LinkOption, info.target) {
		info.linkOptions, err = appendOption(info, info.linkOptions, l, optionPrefix)
		if err != nil {
			result.success = false
			return result, err
		}
	}
	for _, ls := range getList(d.Libraries, info.target) {
		info.libraries, err = appendOption(info, info.libraries, ls, optionPrefix+"l")
		if err != nil {
			result.success = false
			return result, err
		}
	}
	for _, ld := range getList(d.LinkDepend, info.target) {
		info.linkDepends, err = appendOption(info, info.linkDepends, ld, "")
		if err != nil {
			result.success = false
			return result, err
		}
	}
	for _, subninja := range getList(d.SubNinja, info.target) {
		subNinjaList = append(subNinjaList, subninja)
	}

	if err = createOtherRule(info, d.Other, optionPrefix); err != nil {
		return result, err
	}

	files := getList(d.Source, info.target)
	cvfiles := getList(d.Convert_List, info.target)
	testfiles := getList(d.Tests, info.target)

	// sub-directories
	subdirs := getList(d.Subdir, info.target)
	subdirCreateList := []string{}

	for _, s := range subdirs {
		if r, e := build(info, filepath.Join(loaddir, s)); r.success {
			if 0 < len(r.createList) {
				subdirCreateList = append(subdirCreateList, r.createList...)
			}
		} else {
			return r, e
		}
	}

	// pre build files
	if err = createPrebuild(info, loaddir, d.Prebuild); err != nil {
		return result, err
	}

	// create compile list
	createList := []string{}
	if 0 < len(files) {
		var e error
		createList, e = compileFiles(info, loaddir, files)
		if e != nil {
			result.success = false
			return result, e
		}
	}

	switch NowTarget.Type {
	case "library":
		// archive
		if 0 < len(createList) {
			if arname, e := createArchive(info, createList, NowTarget.Name); e == nil {
				result.createList = append(subdirCreateList, arname)
				//fmt.Println(info.archiveOptions+arname+alist)
			} else {
				result.success = false
				return result, e
			}
		} else {
			Warning("There are no files to build in \"%s\".", loaddir)
		}
	case "execute":
		// link program
		if 0 < len(createList) || 0 < len(subdirCreateList) {
			if e := createLink(info, append(createList, subdirCreateList...), NowTarget.Name, NowTarget.Packager); e != nil {
				result.success = false
				return result, e
			}
		} else {
			Warning("There are no files to build in \"%s\".", loaddir)
		}
	case "convert":
		if 0 < len(cvfiles) {
			createConvert(info, loaddir, cvfiles, NowTarget.Name)
		} else {
			Warning("There are no files to convert in \"%s\".", loaddir)
		}
	case "passthrough":
		result.createList = append(subdirCreateList, createList...)
	case "test":
		// unit tests
		if e := createTest(info, testfiles, loaddir); e != nil {
			result.success = false
			return result, e
		}
	default:
		/* NO-OP */
	}

	if verboseMode {
		fmt.Println(pathname+" create list:", len(result.createList))
		for _, rc := range result.createList {
			fmt.Println("    ", rc)
		}
	}
	result.success = true
	return result, nil
}

//
// Retrieves items associated to `targetName`.
//
func getList(block []StringList, targetName string) []string {
	lists := []string{}
	for _, i := range block {
		if (i.Type == "" || i.Type == targetType) && (i.Target == "" || i.Target == targetName) {
			for _, l := range i.List {
				lists = append(lists, l)
			}
			switch {
			case isDebug:
				for _, d := range i.Debug {
					lists = append(lists, d)
				}
			case isDevelop:
				for _, r := range i.Develop {
					lists = append(lists, r)
				}
			case isRelease:
				for _, r := range i.Release {
					lists = append(lists, r)
				}
			case isDevelopRelease:
				for _, r := range i.DevelopRelease {
					lists = append(lists, r)
				}
			case isProduct:
				for _, r := range i.Product {
					lists = append(lists, r)
				}
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
func createArchive(info BuildInfo, createList []string, targetName string) (string, error) {

	var arname string
	switch targetType {
	case "WIN32":
		arname = filepath.Join(info.outputdir, targetName+".lib")
	default:
		arname = filepath.Join(info.outputdir, fmt.Sprintf("lib%s.a", targetName))
	}
	arname = normalizePath(arname)

	archiver, e := info.ExpandVariable("archiver")
	if e != nil {
		return "", e
	}

	cmd := BuildCommand{
		Command:          archiver,
		CommandType:      "ar",
		Args:             info.archiveOptions,
		InFiles:          createList,
		OutFile:          arname,
		NeedCommandAlias: true}
	commandList = append(commandList, cmd)

	return arname, nil
}

//
// link objects
//
func createLink(info BuildInfo, createList []string, targetName string, packager Packager) error {
	trname := filepath.Join(info.outputdir, targetName)
	esuf, ok := info.variables["execute_suffix"]
	if ok {
		trname += esuf
	}
	trname = normalizePath(trname)

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
		pkgname := normalizePath(filepath.Join(outputdir, targetName, packager.Target))
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
		commandList = append(commandList, pkg)
	}
	return nil
}

//
// convert objects
//
func createConvert(info BuildInfo, loaddir string, createList []string, targetName string) {
	cvname := normalizePath(filepath.Join(info.outputdir, targetName))
	converter := info.variables["converter"]

	clist := []string{}
	for _, f := range createList {
		clist = append(clist, normalizePath(loaddir+f))
	}

	cmd := BuildCommand{
		Command:          converter,
		CommandType:      "convert",
		Args:             info.convertOptions,
		InFiles:          clist,
		OutFile:          cvname,
		NeedCommandAlias: true}
	commandList = append(commandList, cmd)
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
		createLink(info, createList, normalizePath(trname), Packager{})
	}
	return nil
}

//
// option
//
func appendOption(info BuildInfo, lists []string, opt string, optionPrefix string) ([]string, error) {
	for _, so := range strings.Split(optionPrefix+opt, " ") {
		var err error
		so, err = info.StrictInterpolate(so)
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
	p := normalizePath(filepath.Join(addDir, ucmd))
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
		if (p.Target == "" || p.Target == info.target) && (p.Type == "" || p.Type == targetType) {
			// register prebuild
			sources := getList(p.Source, info.target)
			if len(sources) == 0 {
				return fmt.Errorf("No sources for command `%s`", p.Name)
			}
			for i, src := range sources {
				if src[0] == '$' {
					sabs, _ := filepath.Abs(filepath.Join(info.outputdir, "output", src[1:]))
					sources[i] = escapeDriveColon(normalizePath(sabs))
				} else if src == "always" {
					sources[i] = src + "|"
				} else {
					if expanded, err := info.Interpolate(src); err == nil {
						sources[i] = normalizePath(filepath.Join(loaddir, expanded))
					} else {
						return err
					}
				}
			}
			ur, ok := info.variables[p.Command]
			if !ok {
				return errors.Errorf("Missing build command \"%s\" (referenced from \"%s\")", p.Command, p.Name)
			}
			mycmd := strings.Replace(normalizePath(filepath.Join(info.outputdir, p.Command)), "/", "_", -1)
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
				outfile = escapeDriveColon(normalizePath(outfile))
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
					outfile = escapeDriveColon(normalizePath(outfile))
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
	}
	return nil
}

// Build command for compiling C, C++...
func compileFiles(info BuildInfo, loaddir string, files []string) (createList []string, e error) {

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
			objdir = normalizePath(filepath.Join(filepath.Dir(srcPath), buildDirectory))
		} else {
			// At this point, `srcPath` is a relative path rooted from `loaddir`
			tf := filepath.Join(loaddir, srcPath)
			objdir = normalizePath(filepath.Join(filepath.Dir(filepath.Join(info.outputdir, srcPath)), buildDirectory))
			dstPathBase = filepath.Base(tf)
			srcPath = tf
		}
		srcPath, _ = filepath.Abs(srcPath)
		sname := escapeDriveColon(normalizePath(srcPath))
		oname := normalizePath(filepath.Join(objdir, dstPathBase+".o"))
		dname := normalizePath(filepath.Join(objdir, dstPathBase+".d"))
		createList = append(createList, oname)

		carg := []string{}
		carg = append(carg, arg1...)
		for _, ca := range info.options {
			switch ca {
			case "$out":
				ca = oname
			case "$dep":
				ca = dname
			case "$in":
				ca = sname
			default:
				/* NO-OP */
			}
			carg = append(carg, ca)
		}
		srcExt := filepath.Ext(srcPath)
		if rule, exists := otherRuleList[srcExt]; exists {
			// custom
			linc := ""
			ldef := ""
			if rule.needInclude == true {
				for _, ii := range info.includes {
					linc += " " + ii
				}
			}
			lopt := ""
			for _, lo := range rule.Options {
				if lo == "$out" {
					lo = oname
				} else if lo == "$dep" {
					lo = dname
				} else if lo == "$in" {
					lo = sname
				}
				lopt += " " + lo
			}
			compiler, ok := info.variables[rule.Compiler]
			if ok == true {
				var err error
				compiler, err = info.Interpolate(compiler)
				if err != nil {
					return []string{}, err
				}
				ocmd := OtherRuleFile{
					rule:     "compile" + srcExt,
					compiler: compiler,
					infile:   sname,
					outfile:  oname,
					include:  linc,
					option:   lopt,
					define:   ldef,
					depend:   ""}
				if rule.NeedDepend == true {
					ocmd.depend = dname
				}
				otherRuleFileList = append(otherRuleFileList, ocmd)
			} else {
				Warning("compiler: Missing a compiler \"%s\" definitions in \"%s\".",
					rule.Compiler,
					normalizePath(filepath.Join(info.mydir, "make.yml")))
			}
		} else {
			// normal
			cmd := BuildCommand{
				Command:          compiler,
				CommandType:      "compile",
				Args:             carg,
				InFiles:          []string{sname},
				OutFile:          oname,
				DepFile:          dname,
				NeedCommandAlias: true}
			if 0 < len(pchFile) {
				cmd.ImplicitDepends = append(cmd.ImplicitDepends, pchFile)
				cmd.Args = append(cmd.Args, "-include-pch", pchFile)
			}
			commandList = append(commandList, cmd)
		}
	}
	return createList, nil
}

// Create pre-compiled header if possible.
func createPCH(info BuildInfo, srcdir string, compiler string) string {
	const pchName = "00-common-prefix.hpp"
	pchSrc := filepath.ToSlash(filepath.Join(srcdir, pchName))
	if !Exists(pchSrc) {
		Verbose ("%s: \"%s\" does not exists.\n", ProgramName, pchSrc)
		return ""
	}
	Verbose ("%s: \"%s\" found.\n", ProgramName, pchSrc)
	dstdir := filepath.Join(info.outputdir, srcdir, buildDirectory)
	pchDst := filepath.ToSlash(filepath.Join(dstdir, pchName+".pch"))
	if verboseMode {
		fmt.Println("Create " + pchDst)
	}
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
	Verbose ("%s: PCH creation command line is \"%s\".", ProgramName, strings.Join(args, " "))
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

func getVariable(info BuildInfo, v Variable) (string, bool) {
	if v.Type != "" && v.Type != targetType {
		return "", false
	}
	if v.Target != "" && v.Target != info.target {
		return "", false
	}
	if v.Build != "" {
		if isDebug && v.Build != "Debug" && v.Build != "debug" {
			return "", false
		}
		if isRelease && v.Build != "Release" && v.Build != "release" {
			return "", false
		}
		if isDevelop && v.Build != "Develop" && v.Build != "develop" {
			return "", false
		}
		if isDevelopRelease && v.Build != "DevelopRelease" && v.Build != "develop_release" {
			return "", false
		}
		if isProduct && v.Build != "Product" && v.Build != "product" {
			return "", false
		}
	}
	return v.Value, true
}

// Creates *.ninja file.
func outputNinja() error {
	Verbose ("%s: Creates \"%s\"", buildNinjaName)

	tPath := NewTransientOutput(buildNinjaName)
	file, err := os.Create(tPath.TempOutput)
	if err != nil {
		return errors.Wrapf (err, "Failed to create temporal output \"%s\"", tPath.TempOutput)
	}
	defer file.Close()
	defer tPath.Abort()
	Verbose ("%s: Creating transient output \"%s\"", ProgramName, tPath.TempOutput)

	// execute build
	if err = outputRules(file); err != nil {
		return errors.Wrapf(err, "Failed to emit rules.")
	}

	for _, bs := range commandList {
		file.WriteString("build " + bs.OutFile + ": " + bs.CommandType)
		for _, f := range bs.InFiles {
			file.WriteString(" $\n  " + f)
		}
		for _, dep := range bs.Depends {
			depstr := escapeDriveColon(dep)
			file.WriteString(" $\n  " + depstr)
		}
		if 0 < len(bs.ImplicitDepends) {
			file.WriteString(" $\n  | " + escapeDriveColon(bs.ImplicitDepends[0]))
			for _, dep := range bs.ImplicitDepends[1:] {
				file.WriteString(" $\n    " + escapeDriveColon(dep))
			}
		}
		if bs.NeedCommandAlias {
			file.WriteString("\n  " + bs.CommandType + " = " + bs.Command + "\n")
		} else {
			file.WriteString("\n")
		}
		if bs.DepFile != "" {
			file.WriteString("  depf = " + bs.DepFile + "\n")
		}
		if len(bs.Args) > 0 {
			tmpLines := make([]string, 0, len(bs.Args))
			for _, t := range bs.Args {
				tmpLines = append(tmpLines, escapeDriveColon(t))
			}
			file.WriteString(fold(tmpLines, 120, "  options ="))
			file.WriteString("\n")
		}
		file.WriteString("  desc = " + bs.OutFile + "\n\n")
	}
	for _, oc := range otherRuleFileList {
		file.WriteString("build " + oc.outfile + ": " + oc.rule + " " + oc.infile + "\n")
		file.WriteString("  compiler = " + oc.compiler + "\n")
		if oc.include != "" {
			file.WriteString("  include = " + oc.include + "\n")
		}
		if oc.option != "" {
			file.WriteString("  option = " + oc.option + "\n")
		}
		if oc.depend != "" {
			file.WriteString("  depf = " + oc.depend + "\n")
		}
		file.WriteString("  desc = " + oc.outfile + "\n\n")
	}

	for _, sn := range subNinjaList {
		file.WriteString("subninja " + sn + "\n")
	}
	if err := file.Close(); err != nil {
		return errors.Wrapf(err,"Closing \"%s\" failed.", file.Name())
	}
	if err := tPath.Commit(); err != nil {
		return errors.Wrapf(err,"Renaming \"%s\" to \"%s\" failed.", tPath.TempOutput, tPath.Output)
	}
	Verbose ("%s: Renaming %s to %s\n", ProgramName, tPath.TempOutput, tPath.Output)
	return nil
}

// Construct a properly folded string from `args`.
func fold(args []string, maxColumns int, prefix string) string {
	lines := make([]string, 0, 8)
	line := ""
	maxcol := maxColumns - len(prefix)
	emptyPrefix := strings.Repeat(" ", len(prefix))
	for _, arg := range args {
		if maxcol < len(line)+1+len(arg) {
			lines = append(lines, prefix+line)
			line = ""
			prefix = emptyPrefix
		}
		line += " " + arg
	}
	if 0 < len(line) {
		lines = append(lines, prefix+line)
	}
	return strings.Join(lines, " $\n")
}

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
		OutputDirectory:    filepath.ToSlash(outputdir),
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

// Normalizes file path.
func normalizePath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
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

// Verbose output if wanted
func Verbose(format string, args ...interface{}) {
	if verboseMode {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

func Warning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s:warning:", ProgramName)
	fmt.Fprintf(os.Stderr, format, args...)
}
