//
//
//
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/kuma777/go-msbuild"
	"gopkg.in/yaml.v2"
)

//
// global variables
//
const (
	cbuildVersion = "1.0.1"
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
)

// The entry point.
func main() {
	gen_msbuild := false
	projdir := ""
	projname := ""

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
	dispVersion := flag.Bool("version", false, "display version")
	flag.Parse()

	exeName := getExeName()

	versionString := cbuildVersion + "(" + runtime.Version() + "/" + runtime.Compiler + ")"
	if *dispVersion {
		fmt.Println(versionString)
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
	if len(ra) > 0 && targetName == "" {
		targetName = ra[0]
	}

	appendRules = map[string]AppendBuild{}
	commandList = []BuildCommand{}
	otherRuleList = map[string]OtherRule{}
	otherRuleFileList = []OtherRuleFile{}
	subNinjaList = []string{}

	if targetName != "" {
		fmt.Println("gobuild: make target: " + targetName)
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
	var r, err = build(buildinfo, "")
	if r.success == false {
		fmt.Printf("%s: error: %s", exeName, err.Error())
		os.Exit(1)
	}

	if nlen := len(commandList) + len(otherRuleFileList); nlen <= 0 {
		fmt.Printf("%s: No commands found.\n", exeName)
		os.Exit(0)
	}
	outputNinja()

	if gen_msbuild {
		outputMSBuild(projdir, projname)
	}
	fmt.Printf("%s: done.\n", exeName)
	os.Exit(0)
}

// Obtains executable name if possible.
func getExeName() string {
	var name = "gobuild"
	if n, err := os.Executable(); err == nil {
		name = n
	}
	return name
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
	if verboseMode == true {
		fmt.Println(pathname + ": start")
	}
	myYaml := filepath.Join(loaddir, "make.yml")
	buf, err := ioutil.ReadFile(myYaml)
	if err != nil {
		e := errors.New(myYaml + ": " + err.Error())
		result.success = false
		return result, e
	}

	var d Data
	err = yaml.Unmarshal(buf, &d)
	if err != nil {
		e := errors.New(myYaml + ": " + err.Error())
		result.success = false
		return result, e
	}

	info.mydir = loaddir
	//
	// select target
	//
	NowTarget, objsSuffix, ok := getTarget(info, d.Target)
	if ok == false {
		e := errors.New("No Target")
		result.success = false
		return result, e
	}
	if info.target == "" {
		info.target = NowTarget.Name
		fmt.Println("gobuild: make target: " + info.target)
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
		val, ok := getVariable(info, v)
		if ok {
			if v.Name == "enable_response" {
				if val == "true" {
					useResponse = true
				} else if val == "false" {
					useResponse = false
				} else {
					fmt.Println(" warning: link_response value [", v.Value, "] is unsupport(true/false)")
				}
			} else if v.Name == "response_newline" {
				if val == "true" {
					responseNewline = true
				} else if val == "false" {
					responseNewline = false
				} else {
					fmt.Println(" warning: link_response value [", v.Value, "] is unsupport(true/false)")
				}
			} else if v.Name == "group_archives" {
				if val == "true" {
					groupArchives = true
				} else if val == "false" {
					groupArchives = false
				} else {
					fmt.Println(" warning: group_archives value [", v.Value, "] is unsupport(true/false)")
				}
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

	info.outputdir = filepath.Join(outputdir, loaddir) + "/"
	objdir := filepath.Clean(filepath.Join(outputdir, loaddir, ".objs"+objsSuffix)) + "/" // Proofs '/' ending

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
			if useRel == false && filepath.IsAbs(pth) == false {
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
	for _, a := range getList(d.Archive_Option, info.target) {
		info.archiveOptions, err = appendOption(info, info.archiveOptions, a, "")
		if err != nil {
			result.success = false
			return result, err
		}
	}
	for _, c := range getList(d.Convert_Option, info.target) {
		info.convertOptions, err = appendOption(info, info.convertOptions, c, "")
		if err != nil {
			result.success = false
			return result, err
		}
	}
	for _, l := range getList(d.Link_Option, info.target) {
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
	for _, ld := range getList(d.Link_Depend, info.target) {
		info.linkDepends, err = appendOption(info, info.linkDepends, ld, "")
		if err != nil {
			result.success = false
			return result, err
		}
	}
	for _, subninja := range getList(d.SubNinja, info.target) {
		subNinjaList = append(subNinjaList, subninja)
	}

	err = createOtherRule(info, d.Other, optionPrefix)
	if err != nil {
		return result, err
	}

	files := getList(d.Source, info.target)
	cvfiles := getList(d.Convert_List, info.target)
	testfiles := getList(d.Tests, info.target)

	// sub-directories
	subdirs := getList(d.Subdir, info.target)
	subdirCreateList := []string{}
	for _, s := range subdirs {
		sd := loaddir + s
		var r, e = build(info, sd)
		if r.success == false {
			return r, e
		}
		if len(r.createList) > 0 {
			subdirCreateList = append(subdirCreateList, r.createList...)
		}
	}

	// pre build files
	err = createPrebuild(info, loaddir, d.Prebuild)
	if err != nil {
		return result, err
	}

	// create compile list
	createList := []string{}
	if len(files) > 0 {
		var e error
		createList, e = compileFiles(info, objdir, loaddir, files)
		if e != nil {
			result.success = false
			return result, e
		}
	}

	if NowTarget.Type == "library" {
		// archive
		if len(createList) > 0 {
			arname, e := createArchive(info, createList, NowTarget.Name)
			if e != nil {
				result.success = false
				return result, e
			}
			result.createList = append(subdirCreateList, arname)
			//fmt.Println(info.archiveOptions+arname+alist)
		} else {
			fmt.Println("There are no files to build.", loaddir)
		}
	} else if NowTarget.Type == "execute" {
		// link program
		if len(createList) > 0 || len(subdirCreateList) > 0 {
			e := createLink(info, append(createList, subdirCreateList...), NowTarget.Name, NowTarget.Packager)
			if e != nil {
				result.success = false
				return result, e
			}
		} else {
			fmt.Println("There are no files to build.", loaddir)
		}
	} else if NowTarget.Type == "convert" {
		if len(cvfiles) > 0 {
			createConvert(info, loaddir, cvfiles, NowTarget.Name)
		} else {
			fmt.Println("There are no files to convert.", loaddir)
		}
	} else if NowTarget.Type == "passthrough" {
		result.createList = append(subdirCreateList, createList...)
	} else if NowTarget.Type == "test" {
		// unit tests
		e := createTest(info, testfiles, loaddir)
		if e != nil {
			result.success = false
			return result, e
		}
	} else {
		//
		// other...
		//
	}
	if verboseMode == true {
		fmt.Println(pathname+" create list:", len(result.createList))
		for _, rc := range result.createList {
			fmt.Println("    ", rc)
		}
	}
	result.success = true
	return result, nil
}

//
//
//
func getList(block []StringList, targetName string) []string {
	lists := []string{}
	for _, i := range block {
		if (i.Type == "" || i.Type == targetType) && (i.Target == "" || i.Target == targetName) {
			for _, l := range i.List {
				lists = append(lists, l)
			}
			if isDebug == true {
				for _, d := range i.Debug {
					lists = append(lists, d)
				}
				continue
			}
			if isDevelop == true {
				for _, r := range i.Develop {
					lists = append(lists, r)
				}
				continue
			}
			if isRelease == true {
				for _, r := range i.Release {
					lists = append(lists, r)
				}
				continue
			}
			if isDevelopRelease == true {
				for _, r := range i.DevelopRelease {
					lists = append(lists, r)
				}
				continue
			}
			if isProduct == true {
				for _, r := range i.Product {
					lists = append(lists, r)
				}
				continue
			}
		}
	}
	return lists
}

//
func getReplacedVariable(info BuildInfo, name string) (string, error) {
	str, ok := info.variables[name]
	if ok == false {
		return "", errors.New("not found variable: " + name)
	}
	return info.Interpolate(str)
}

//
// archive objects
//
func stringToReplacedList(info BuildInfo, str string) ([]string, error) {
	sl := strings.Split(str, " ")
	for i, s := range sl {
		expanded, err := info.Interpolate(s)
		if err != nil {
			return []string{}, err
		}
		sl [i] = expanded
	}
	return sl, nil
}

//
// link objects
//
func createArchive(info BuildInfo, createList []string, targetName string) (string, error) {

	arname := info.outputdir
	if targetType == "WIN32" {
		arname += targetName + ".lib"
	} else {
		arname += "lib" + targetName + ".a"
	}
	arname = filepath.ToSlash(filepath.Clean(arname))

	archiver, e := getReplacedVariable(info, "archiver")
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
// convert objects
//
func createLink(info BuildInfo, createList []string, targetName string, packager Packager) error {
	trname := info.outputdir + targetName
	esuf, ok := info.variables["execute_suffix"]
	if ok {
		trname += esuf
	}
	trname = filepath.ToSlash(filepath.Clean(trname))

	linker, e := getReplacedVariable(info, "linker")
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
		pkgname := filepath.ToSlash(filepath.Clean(outputdir + "/" + targetName + "/" + packager.Target))
		pkgr, e := getReplacedVariable(info, "packager")
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
// unit tests
//
func createConvert(info BuildInfo, loaddir string, createList []string, targetName string) {
	cvname := info.outputdir + targetName
	cvname = filepath.ToSlash(filepath.Clean(cvname))
	converter := info.variables["converter"]

	clist := []string{}
	for _, f := range createList {
		clist = append(clist, filepath.ToSlash(filepath.Clean(loaddir+f)))
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
// option
//
func createTest(info BuildInfo, createList []string, loaddir string) error {
	carg := append(info.includes, info.defines...)
	for _, ca := range info.options {
		if ca != "$out" && ca != "$dep" && ca != "$in" && ca != "-c" { // FIXME.
			carg = append(carg, ca)
		}
	}
	objdir := info.outputdir + ".objs/"

	for _, f := range createList {
		// first, compile a test driver
		createList, _ = compileFiles(info, objdir, loaddir, []string{f})

		// then link it as an executable (test_aaa.cpp -> test_aaa)
		trname := strings.TrimSuffix(f, filepath.Ext(f))
		esuf, ok := info.variables["execute_suffix"]
		if ok {
			trname += esuf
		}
		trname = filepath.ToSlash(filepath.Clean(trname))
		createLink(info, createList, trname, Packager{})

	}
	return nil
}

//
// target
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
				if info.target == t.By_Target {
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
	p := filepath.ToSlash(filepath.Clean(addDir + ucmd))
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
			// regist prebuild
			srlist := getList(p.Source, info.target)
			if len(srlist) == 0 {
				e := errors.New("build command: " + p.Name + " is empty source.")
				return e
			}
			for i, src := range srlist {
				if src[0] == '$' {
					sabs, _ := filepath.Abs(info.outputdir + "output/" + src[1:])
					sabs = strings.Replace(sabs, ":", "$:", 1)
					srlist[i] = filepath.ToSlash(filepath.Clean(sabs))
				} else if src == "always" {
					srlist[i] = src + "|"
				} else {
					var err error
					src, err = info.Interpolate(src)
					if err != nil {
						return err
					}
					srlist[i] = filepath.ToSlash(filepath.Clean(loaddir + src))
				}
			}
			ur, ok := info.variables[p.Command]
			if ok == false {
				e := errors.New("build command: <" + p.Command + "> is not found.(use by " + p.Name + ")")
				return e
			}
			mycmd := strings.Replace(filepath.ToSlash(filepath.Clean(info.outputdir+p.Command)), "/", "_", -1)
			deps := []string{}
			_, af := appendRules[mycmd]
			if af == false {
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
				outfile, _ := filepath.Abs(info.outputdir + pn)
				outfile = strings.Replace(filepath.ToSlash(filepath.Clean(outfile)), ":", "$:", -1)
				cmd := BuildCommand{
					Command:          p.Command,
					CommandType:      mycmd,
					Depends:          deps,
					InFiles:          srlist,
					OutFile:          outfile,
					NeedCommandAlias: false}
				commandList = append(commandList, cmd)
			} else {
				ext := p.Name[1:] //filepath.Ext(p.Name)
				for _, src := range srlist {
					dst := filepath.Base(src)
					next := filepath.Ext(src)
					if next != "" {
						dst = dst[0:len(dst)-len(next)] + ext
					} else {
						dst += ext
					}
					outfile, _ := filepath.Abs(info.outputdir + "output/" + dst)
					outfile = strings.Replace(filepath.ToSlash(filepath.Clean(outfile)), ":", "$:", -1)
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

// Creates pre-compiled header if `precompile.hpp` exists.
func compileFiles(info BuildInfo, objdir string, loaddir string, files []string) (createList []string, e error) {

	compiler, e := getReplacedVariable(info, "compiler")
	if e != nil {
		return []string{}, e
	}

	createList = append(createList, createPCH(info, objdir, loaddir, compiler)...)
	arg1 := append(info.includes, info.defines...)

	for _, f := range files {
		of := f
		if f[0] == '$' {
			if strings.HasPrefix(f, "$target/") {
				of = strings.Replace(of, "$target/", "/."+info.target+"/", 1)
			} else {
				of = f[1:]
			}
			f = info.outputdir + of
		} else {
			f = loaddir + f
		}
		f, _ = filepath.Abs(f)
		sname := strings.Replace(filepath.ToSlash(filepath.Clean(f)), ":", "$:", -1)
		oname := filepath.ToSlash(filepath.Clean(objdir + of + ".o"))
		dname := filepath.ToSlash(filepath.Clean(objdir + of + ".d"))
		createList = append(createList, oname)

		carg := []string{}
		carg = append(carg, arg1...)
		for _, ca := range info.options {
			if ca == "$out" {
				ca = oname
			} else if ca == "$dep" {
				ca = dname
			} else if ca == "$in" {
				ca = sname
			}
			carg = append(carg, ca)
		}
		ext := filepath.Ext(f)
		rule, ok := otherRuleList[ext]
		if ok == false {
			// normal
			cmd := BuildCommand{
				Command:          compiler,
				CommandType:      "compile",
				Args:             carg,
				InFiles:          []string{sname},
				OutFile:          oname,
				DepFile:          dname,
				NeedCommandAlias: true}
			commandList = append(commandList, cmd)
		} else {
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
					rule:     "compile" + ext,
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
				fmt.Println("compiler:", rule.Compiler, "is not found. in ["+info.mydir+"make.yml].")
			}
		}
	}

	return createList, nil
}

func createPCH(info BuildInfo, dstdir string, srcdir string, compiler string) []string {
	const pchName = "00-common-prefix.hpp"
	pchSrc := filepath.Join(srcdir, pchName)
	if !Exists(pchSrc) {
		if verboseMode {
			fmt.Println(pchSrc + " does not exists.")
		}
		return []string{}
	}
	if verboseMode {
		fmt.Println(pchSrc + " found.")
	}
	pchDst := filepath.Join(dstdir, pchName+".pch")
	fmt.Println("Create " + pchDst)
	// PCH source found.
	return []string{}
}

//
// other rule
//
func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

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
		if ok == false {

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
				NeedDepend:  ot.needDependend}
		} else {
			rule.Options = append(rule.Options, olist...)
		}
		otherRuleList[ext] = rule
	}
	return nil
}

//
func checkType(vlist []Variable) string {
	for _, v := range vlist {
		if v.Name == "default_type" {
			return v.Value
		}
	}
	return "default"
}

//
// build main
//
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

//
// Creates *.ninja file.
//
func outputNinja() {
	exeName := getExeName()
	if verboseMode {
		fmt.Println("output " + buildNinjaName)
	}
	tPath := NewTransientOutput(buildNinjaName)
	file, err := os.Create(tPath.TempOutput)
	if err != nil {
		fmt.Println("gobuild: error:", err.Error())
		os.Exit(1)
	}
	defer tPath.Abort()
	if verboseMode {
		fmt.Printf("%s: Creating transient output: %s\n", exeName, tPath.TempOutput)
	}
	// execute build
	outputRules(file)

	for _, bs := range commandList {
		file.WriteString("build " + bs.OutFile + ": " + bs.CommandType)
		for _, f := range bs.InFiles {
			file.WriteString(" $\n  " + f)
		}
		for _, dep := range bs.Depends {
			depstr := strings.Replace(dep, ":", "$:", 1)
			file.WriteString(" $\n  " + depstr)
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
			file.WriteString("  options =")
			for i, o := range bs.Args {
				if i&3 == 3 {
					file.WriteString(" $\n   ")
				}
				ostr := strings.Replace(o, ":", "$:", 1)
				file.WriteString(" " + ostr)
			}
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
		tPath.Abort()
		fmt.Printf("%s: Close failed.\n", exeName)
		os.Exit(1)
	}
	if err := tPath.Commit(); err != nil {
		fmt.Printf("%s: Renaming %s to %s failed.\n", exeName, tPath.TempOutput, tPath.Output)
		os.Exit(1)
	}
	if verboseMode {
		fmt.Printf("%s: Renaming %s to %s\n", exeName, tPath.TempOutput, tPath.Output)
	}
}

// Emits common rules.
func outputRules(file *os.File) {
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
    command = $compile $options -x c++header -o $out $in
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
{{- if .UseResponse}}
    description = Linking: $desc
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
    command = $link $options -o $out $in
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

	err := tmpl.Execute(file, ctx)
	if err != nil {
		panic(err)
	}
}

//
// Creates *.vcxproj (for VisualStudio).
//
func outputMSBuild(outdir, projname string) {
	var targets []string

	for _, command := range commandList {
		if command.CommandType != "compile" {
			continue
		}

		for _, infile := range command.InFiles {
			targets = append(targets, strings.Replace(infile, "$:", ":", 1))
		}
	}

	msbuild.ExportProject(targets, outdir, projname)
}

//
//
