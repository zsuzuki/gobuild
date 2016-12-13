//
//
//
package main

import (
    "fmt"
    "flag"
    "gopkg.in/yaml.v2"
    "io/ioutil"
    "path/filepath"
    "os"
    "strings"
    //"os/exec"
    //"unsafe"
    "github.com/kuma777/go-msbuild"
)

//
// data structures
//
type Target struct {
    Name string
    Type string
    By_Target string
}
type StringList struct {
    Type string
    Target string
    Debug []string `yaml:",flow"`
    Release []string `yaml:",flow"`
    Develop []string `yaml:",flow"`
    Product []string `yaml:",flow"`
    List []string `yaml:",flow"`
}
type Variable struct {
    Name string
    Value string
    Type string
    Target string
    Build string
}
type Build struct {
    Name string
    Command string
    Target string
    Type string
    Source []StringList `yaml:",flow"`
}

type Other struct {
    Ext string
    Command string
    Description string
    Need_Depend bool
    Type string
    Option []StringList `yaml:",flow"`
}

type Data struct {
    Target []Target `yaml:",flow"`
    Include []StringList `yaml:",flow"`
    Variable []Variable `yaml:",flow"`
    Define []StringList `yaml:",flow"`
    Option []StringList `yaml:",flow"`
    Archive_Option []StringList `yaml:",flow"`
    Convert_Option []StringList `yaml:",flow"`
    Link_Option []StringList `yaml:",flow"`
    Link_Depend []StringList `yaml:",flow"`
    Libraries []StringList `yaml:",flow"`
    Prebuild []Build `yaml:",flow"`
    Postbuild []Build `yaml:",flow"`
    Source []StringList `yaml:",flow"`
    Convert_List []StringList `yaml:",flow"`
    Subdir []StringList `yaml:",flow"`
    Other []Other `yaml:",flow"`
}

//
// error
//

type MyError struct {
    str string
}
func (m MyError) Error() string {
    return m.str
}

//
// build information
//

//
type OtherRule struct {
    compiler string
    cmd string
    title string
    option []string
    need_inc bool
    need_opt bool
    need_def bool
    need_dep bool
}

type OtherRuleFile struct {
    rule string
    compiler string
    infile string
    outfile string
    include string
    option string
    define string
    depend string
}

//
type BuildCommand struct {
    cmd string
    cmdtype string
    cmdalias string
    args []string
    infiles []string
    outfile string
    depfile string
    depends []string
    need_cmd_alias bool
}

//
type BuildResult struct {
    success bool
    create_list []string
}

//
type BuildInfo struct {
    variables map[string] string
    includes []string
    defines []string
    options []string
    archive_options []string
    convert_options []string
    link_options []string
    link_depends []string
    libraries []string
    select_target string
    target string
    outputdir string
    subdir []string
    mydir string
}

//
// global variables
//
var (
    isDebug bool
    isRelease bool
    isProduct bool
    isDevelop bool
    target_type string
    target_name string
    toplevel bool
    outputdir string
    outputdir_set bool
    append_rules map[string] string
    other_rule_list map[string] OtherRule

    command_list []BuildCommand
    other_rule_file_list []OtherRuleFile

    verboseMode bool
    useResponse bool
)

//
//
// build functions
//
//

//
//
//
func getList(block []StringList,target_name string) []string {
    lists := [] string{}
    for _,i := range block {
        if (i.Type == "" || i.Type == target_type) && (i.Target == "" || i.Target == target_name) {
            for _,l := range i.List {
                lists = append(lists,l)
            }
            if isDebug == true {
                for _,d := range i.Debug {
                    lists = append(lists,d)
                }
            } else if isDevelop == true {
                for _,r := range i.Develop {
                    lists = append(lists,r)
                }
            } else if isRelease == true {
                for _,r := range i.Release {
                    lists = append(lists,r)
                }
            } else if isProduct == true {
                for _,r := range i.Product {
                    lists = append(lists,r)
                }
            }
        }
    }
    return lists
}

//
// archive objects
//
func create_archive(info BuildInfo,create_list []string,target_name string) (string,error) {

    arname := info.outputdir
    if target_type == "WIN32" {
        arname += target_name + ".lib"
    } else {
        arname += "lib" + target_name + ".a"
    }
    arname = filepath.ToSlash(filepath.Clean(arname))

    archiver := info.variables["archiver"]
    avi := strings.Index(archiver,"${")
    if avi != -1 {
        var e error
        archiver,e = replace_variable(info,archiver,avi,true)
        if e != nil {
            return "",e
        }
    }

    cmd := BuildCommand{
        cmd : archiver,
        cmdtype : "ar",
        args : info.archive_options,
        infiles : create_list,
        outfile : arname,
        need_cmd_alias : true }
    command_list = append(command_list,cmd)

    return arname,nil
}

//
// link objects
//
func create_link(info BuildInfo,create_list []string,target_name string) error {
    trname := info.outputdir + target_name
    esuf, ok := info.variables["execute_suffix"]
    if ok {
        trname += esuf
    }
    trname = filepath.ToSlash(filepath.Clean(trname))

    linker := info.variables["linker"]
    lvi := strings.Index(linker,"${")
    if lvi != -1 {
        var e error
        linker,e = replace_variable(info,linker,lvi,true)
        if e != nil {
            return e
        }
    }

    options := []string{}
    for _,lo := range info.link_options {
        lo = strings.Replace(lo,"$out",trname,-1)
        options = append(options,lo)
    }
    options = append(options,info.libraries...)

    cmd := BuildCommand{
        cmd : linker,
        cmdtype : "link",
        args : options,
        infiles : create_list,
        outfile : trname,
        depends : info.link_depends,
        need_cmd_alias : true }
    command_list = append(command_list,cmd)
    //fmt.Println("-o " + NowTarget.Name + flist)
    return nil
}

//
// convert objects
//
func create_convert(info BuildInfo,loaddir string,create_list []string,target_name string) {
    cvname := info.outputdir + target_name
    cvname = filepath.ToSlash(filepath.Clean(cvname))
    converter := info.variables["converter"]

    clist := []string{}
    for _,f := range create_list {
        clist = append(clist,filepath.ToSlash(filepath.Clean(loaddir+f)))
    }

    cmd := BuildCommand{
        cmd : converter,
        cmdtype : "convert",
        args : info.convert_options,
        infiles : clist,
        outfile : cvname,
        need_cmd_alias : true }
    command_list = append(command_list,cmd)
}

//
// option
//
func append_option(info BuildInfo,lists []string,opt string,opt_pre string) ([]string, error) {
    sl := strings.Split(opt_pre+opt," ")
    for _,so := range sl {
        si := strings.Index(so,"${")
        if si != -1 {
            var e error
            so,e = replace_variable(info,so,si,false)
            if e != nil {
                return lists,e
            }
        }
        if strings.Index(so," ") != -1 {
            so = "\""+so+"\""
        }
        lists = append(lists,so)
    }
    return lists,nil
}

//
// target
//
func get_target(info BuildInfo,tlist []Target) (Target,string,bool) {
    if info.select_target != "" {
        // search target
        for _,t := range tlist {
            if info.select_target == t.Name {
                return t,"_"+info.select_target,true
            }
        }

    } else {
        if info.target != "" {

            // search by_target
            for _,t := range tlist {
                if info.target == t.By_Target {
                    return t,"_"+info.target,true
                }
            }
            // search target
            for _,t := range tlist {
                if info.target == t.Name {
                    return t,"_"+info.target,true
                }
            }
        }
        if len(tlist) > 0 {
            t := tlist[0]
            if info.target == "" {
                return t,"_"+t.Name,true
            } else {
                return t,"",true
            }
        }
    }
    return Target{},"",false
}

//
func replace_path(value string,add_dir string) (string, string) {
    url := strings.Split(value," ")
    ucmd := url[0]
    if ucmd[0] == '$' {
        ucmd = ucmd[1:]
    }
    p := filepath.ToSlash(filepath.Clean(add_dir+ucmd))
    result := p
    for i,uu := range url {
        if i > 0 {
            result += " "+uu
        }
    }
    return result,p
}

//
func replace_variable(info BuildInfo,str string,start int,no_error bool) (string, error) {
    src := strings.Split(str[start+2:],"${")
    ret := str[:start]
    for _,s := range src {
        br := strings.Index(s,"}")
        if br == -1 {
            e := MyError{ str: "variable not close ${name}. \"${"+s+"\" in ["+info.mydir+"make.yml]." }
            return "",e
        }
        vname := s[:br]
        v,ok := info.variables[vname]
        if ok == false {
            if no_error {
                v = ""
            } else {
                e := MyError{ str: "variable <"+vname+"> is not found in ["+info.mydir+"make.yml]." }
                return "",e
            }
        }
        ret += v + s[br+1:]
    }
    return ret,nil
}

//
// pre build
//
func create_prebuild(info BuildInfo,loaddir string,plist []Build) error {
    for _,p := range plist {
        if (p.Target == "" || p.Target == info.target) && (p.Type == "" || p.Type == target_type) {
            // regist prebuild
            srlist := getList(p.Source,info.target)
            if len(srlist) == 0 {
                e := MyError{ str : "build command: "+p.Name+" is empty source." }
                return e
            }
            for i,src := range srlist {
                if src[0] == '$' {
                    sabs,_ := filepath.Abs(info.outputdir+"output/"+src[1:len(src)])
                    sabs = strings.Replace(sabs,":","$:",1)
                    srlist[i] = filepath.ToSlash(filepath.Clean(sabs))
                } else if src == "always" {
                    srlist[i] = src+"|"
                } else {
                    srlist[i] = filepath.ToSlash(filepath.Clean(loaddir+src))
                }
            }
            ur,ok := info.variables[p.Command]
            if ok == false {
                e := MyError{ str : "build command: <"+p.Command+"> is not found.(use by "+p.Name+")"}
                return e
            }
            mycmd := strings.Replace(filepath.ToSlash(filepath.Clean(info.outputdir+p.Command)),"/","_",-1)
            deps := []string{}
            _,af := append_rules[mycmd]
            if af == false {
                ur = strings.Replace(ur,"${selfdir}",loaddir,-1)
                ev := strings.Index(ur,"${")
                if ev != -1 {
                    var e error
                    ur,e = replace_variable(info,ur,ev,false)
                    if e != nil {
                        return e
                    }
                }

                if ur[0] == '$' {
                    r, d := replace_path(ur,info.outputdir)
                    abs,_ := filepath.Abs(d)
                    d = filepath.ToSlash(abs)
                    deps = append(deps,d)
                    ur = r
                } else if strings.HasPrefix(ur,"../") || strings.HasPrefix(ur,"./") {
                    r, d := replace_path(ur,loaddir)
                    deps = append(deps,d)
                    ur = r
                }
                ur = strings.Replace(ur,"$target",info.target,-1)
                append_rules[mycmd] = ur
            }

            if p.Name[0] != '$' || strings.HasPrefix(p.Name,"$target/") {
                pn := p.Name
                if pn[0] == '$' {
                    pn = strings.Replace(pn,"$target/","/."+info.target+"/",1)
                }
                outfile,_ := filepath.Abs(info.outputdir+pn)
                outfile = strings.Replace(filepath.ToSlash(filepath.Clean(outfile)),":","$:",-1)
                cmd := BuildCommand{
                    cmd : p.Command,
                    cmdtype : mycmd,
                    depends : deps,
                    infiles : srlist,
                    outfile : outfile,
                    need_cmd_alias : false }
                command_list = append(command_list,cmd)
            } else {
                ext := p.Name[1:]//filepath.Ext(p.Name)
                for _,src := range srlist {
                    dst := filepath.Base(src)
                    next := filepath.Ext(src)
                    if next != "" {
                        dst = dst[0:len(dst)-len(next)]+ext
                    } else {
                        dst += ext
                    }
                    outfile,_ := filepath.Abs(info.outputdir+"output/"+dst)
                    outfile = strings.Replace(filepath.ToSlash(filepath.Clean(outfile)),":","$:",-1)
                    cmd := BuildCommand{
                        cmd : p.Command,
                        cmdtype : mycmd,
                        depends : deps,
                        infiles : []string{ src },
                        outfile : outfile,
                        need_cmd_alias : false }
                    command_list = append(command_list,cmd)
                }
            }
        }
    }
    return nil
}

//
// compile files
//
func compile_files(info BuildInfo,objdir string,loaddir string,files []string) (create_list []string,e error) {

    compiler := info.variables["compiler"]
    cvi := strings.Index(compiler,"${")
    if cvi != -1 {
        compiler,e = replace_variable(info,compiler,cvi,true)
        if e != nil {
            return []string{},e
        }
    }

    arg1 := append(info.includes,info.defines...)

    for _,f := range files {
        of := f
        if f[0] == '$' {
            if strings.HasPrefix(f,"$target/") {
                of = strings.Replace(of,"$target/","/."+info.target+"/",1)
            } else {
                of = f[1:]
            }
            f = info.outputdir + of
        } else {
            f = loaddir+f
        }
        f,_ = filepath.Abs(f)
        sname := filepath.ToSlash(filepath.Clean(f))
        sname = strings.Replace(sname,":","$:",-1)
        oname := filepath.ToSlash(filepath.Clean(objdir+of+".o"))
        dname := filepath.ToSlash(filepath.Clean(objdir+of+".d"))
        create_list = append(create_list,oname)

        carg := []string{}
        carg = append(carg,arg1...)
        for _,ca := range info.options {
            if ca == "$out" {
                ca = oname
            } else if ca == "$dep" {
                ca = dname
            } else if ca == "$in" {
                ca = sname
            }
            carg = append(carg,ca)
        }
        ext := filepath.Ext(f)
        rule, ok := other_rule_list[ext]
        if ok == false {
            // normal
            cmd := BuildCommand{
                cmd : compiler,
                cmdtype : "compile",
                args : carg,
                infiles : []string{ sname },
                outfile : oname,
                depfile : dname,
                need_cmd_alias : true }
            command_list = append(command_list,cmd)
        } else {
            // custom
            linc := ""
            ldef := ""
            if rule.need_inc == true {
                for _, ii := range info.includes {
                    linc += " "+ii
                }
            }
            lopt := ""
            for _,lo := range rule.option {
                if lo == "$out" {
                    lo = oname
                } else if lo == "$dep" {
                    lo = dname
                } else if lo == "$in" {
                    lo = sname
                }
                lopt += " " + lo
            }
            compiler,ok := info.variables[rule.compiler]
            if ok == true {
                cvi = strings.Index(compiler,"${")
                if cvi != -1 {
                    var e error
                    compiler,e = replace_variable(info,compiler,cvi,true)
                    if e != nil {
                        return []string{},e
                    }
                }
                ocmd := OtherRuleFile{
                    rule : "compile"+ext[1:],
                    compiler : compiler,
                    infile : sname,
                    outfile : oname,
                    include : linc,
                    option : lopt,
                    define : ldef,
                    depend : "" }
                if rule.need_dep == true {
                    ocmd.depend = dname
                }
                other_rule_file_list = append(other_rule_file_list,ocmd)
            } else {
                fmt.Println("compiler:",rule.compiler,"is not found. in ["+info.mydir+"make.yml].")
            }
        }
    }

    return create_list,nil
}

//
// other rule
//
func create_other_rules(info BuildInfo,olist []Other,opt_pre string) error {
    for _, ot := range olist {
        if ot.Type != "" && ot.Type != target_type {
            continue
        }

        ext := ot.Ext

        olist := []string{}
        for _,o := range getList(ot.Option,info.target) {
            var e error
            olist,e = append_option(info,olist,o,opt_pre)
            if e != nil {
                return e
            }
        }

        need_inc := false
        need_opt := false
        need_def := false
        rule, ok := other_rule_list[ ext ]
        if ok == false {

            // no exist rule
            cmdl := strings.Split(ot.Command," ")
            compiler := ""

            cmdline := "$compiler"
            for i,c := range cmdl {
                if i == 0 {
                    compiler = c
                } else if c[0] == '@' {
                    switch c {
                    case "@include": need_inc = true
                    case "@option": need_opt = true
                    case "@define": need_def = true
                    }
                    cmdline += " $" + c[1:]
                } else {
                    cmdline += " "+c
                }
            }

            rule = OtherRule{
                compiler : compiler,
                cmd : cmdline,
                title : ot.Description,
                option : olist,
                need_inc : need_inc,
                need_opt : need_opt,
                need_def : need_def,
                need_dep : ot.Need_Depend }
        } else {
            rule.option = append(rule.option,olist...)
        }
        other_rule_list[ ext ] = rule
    }
    return nil
}

//
func check_type(vlist []Variable) string {
    for _,v := range vlist {
        if v.Name == "default_type" {
            return v.Value
        }
    }
    return "default"
}

//
func get_variable(info BuildInfo,v Variable) (string, bool) {
    if v.Type != "" && v.Type != target_type {
        return "",false
    }
    if v.Target != "" && v.Target != info.target {
        return "",false
    }
    if v.Build != "" {
        if isDebug && v.Build != "Debug" && v.Build != "debug" {
            return "",false
        } else if isRelease && v.Build != "Release" && v.Build != "release" {
            return "",false
        } else if isDevelop && v.Build != "Develop" && v.Build != "develop" {
            return "",false
        } else if isProduct && v.Build != "Product" && v.Build != "product" {
            return "",false
        }
    }
    return v.Value,true
}




//
// build main
//
func build(info BuildInfo,pathname string) (result BuildResult,err error) {
    loaddir := pathname
    if loaddir == "" {
        loaddir = "./"
    } else {
        loaddir += "/"
    }
    if verboseMode == true {
        fmt.Println(pathname+": start")
    }
    my_yaml := loaddir+"make.yml"
    buf, err := ioutil.ReadFile(my_yaml)
    if err != nil {
        e := MyError{ str : my_yaml + ": " + err.Error() }
        result.success = false
        return result,e
    }

    var d Data
    err = yaml.Unmarshal(buf, &d)
    if err != nil {
        e := MyError{ str : my_yaml + ": " + err.Error() }
        result.success = false
        return result,e
    }

    info.mydir = loaddir
    //
    // select target
    //
    NowTarget,objs_suffix,ok := get_target(info,d.Target)
    if ok == false {
        e := MyError{ str : "No Target" }
        result.success = false
        return result,e
    }
    if info.target == "" {
        info.target = NowTarget.Name
        fmt.Println("gobuild: make target: "+info.target)
    }
    info.select_target = ""

    if toplevel == true && target_type == "default" {
        target_type = check_type(d.Variable)
    }
    toplevel = false
    //
    // get rules
    //
    for _,v := range d.Variable {
        val,ok := get_variable(info,v)
        if ok {
            if v.Name == "enable_response" {
                if val == "true" {
                    useResponse = true
                } else if val == "false" {
                    useResponse = false
                } else {
                    fmt.Println(" warning: link_response value [",v.Value,"] is unsupport(true/false)")
                }
            }
            info.variables[v.Name] = val
        }
    }
    opt_pre := info.variables["option_prefix"]
    if outputdir_set == false {
        outputdir += "/" + target_type + "/"
        if isProduct {
            outputdir += "Product"
        } else if isDevelop {
            outputdir += "Develop"
        } else if isRelease {
            outputdir += "Release"
        } else {
            outputdir += "Debug"
        }
        outputdir_set = true
    }

    info.outputdir = outputdir + "/" + loaddir
    objdir := outputdir + "/" + loaddir + ".objs"+objs_suffix+"/"

    for _,i := range getList(d.Include,info.target) {
        if strings.HasPrefix(i,"$output") {
            i = filepath.Clean(info.outputdir + "output" + i[7:])
        } else if filepath.IsAbs(i) == false {
            i = filepath.Clean(loaddir + i)
        }
        ii := strings.Index(i,"${")
        if ii != -1 {
            i,err = replace_variable(info,i,ii,false)
            if err != nil {
                result.success = false
                return result,err
            }
        }
        if strings.Index(i," ") != -1 {
            i = "\""+i+"\""
        }
        info.includes = append(info.includes,opt_pre + "I" + filepath.ToSlash(i))
    }
    for _,d := range getList(d.Define,info.target) {
        info.defines = append(info.defines,opt_pre + "D" + d)
    }
    for _,o := range getList(d.Option,info.target) {
        info.options,err = append_option(info,info.options,o,opt_pre)
        if err != nil {
            result.success = false
            return result,err
        }
    }
    for _,a := range getList(d.Archive_Option,info.target) {
        info.archive_options,err = append_option(info,info.archive_options,a,"")
        if err != nil {
            result.success = false
            return result,err
        }
    }
    for _,c := range getList(d.Convert_Option,info.target) {
        info.convert_options,err = append_option(info,info.convert_options,c,"")
        if err != nil {
            result.success = false
            return result,err
        }
    }
    for _,l := range getList(d.Link_Option,info.target) {
        info.link_options,err = append_option(info,info.link_options,l,opt_pre)
        if err != nil {
            result.success = false
            return result,err
        }
    }
    for _,ls := range getList(d.Libraries,info.target) {
        info.libraries,err = append_option(info,info.libraries,ls,opt_pre+"l")
        if err != nil {
            result.success = false
            return result,err
        }
    }
    for _,ld := range getList(d.Link_Depend,info.target) {
        info.link_depends,err = append_option(info,info.link_depends,ld,"")
        if err != nil {
            result.success = false
            return result,err
        }
    }

    err = create_other_rules(info,d.Other,opt_pre)
    if err != nil {
        return result,err
    }

    files := getList(d.Source,info.target)
    cvfiles := getList(d.Convert_List,info.target)

    // sub-directories
    subdirs := getList(d.Subdir,info.target)
    subdir_create_list := []string{}
    for _,s := range subdirs {
        sd := loaddir+s
        var r,e = build(info,sd)
        if r.success == false {
            return r,e
        }
        if len(r.create_list) > 0 {
            subdir_create_list = append(subdir_create_list,r.create_list...)
        }
    }

    // pre build files
    err = create_prebuild(info,loaddir,d.Prebuild)
    if err != nil {
        return result,err
    }

    // create compile list
    create_list := []string{}
    if len(files) > 0 {
        var e error
        create_list,e = compile_files(info,objdir,loaddir,files)
        if e != nil {
            result.success = false
            return result,e
        }
    }

    if NowTarget.Type == "library" {
        // archive
        if len(create_list) > 0 {
            arname, e := create_archive(info,create_list,NowTarget.Name)
            if e != nil {
                result.success = false
                return result,e
            }
            result.create_list = append(subdir_create_list,arname)
            //fmt.Println(info.archive_options+arname+alist)
        } else {
            fmt.Println("There are no files to build.",loaddir)
        }
    } else if NowTarget.Type == "execute" {
        // link program
        if len(create_list) > 0 || len(subdir_create_list) > 0 {
            e := create_link(info,append(create_list,subdir_create_list...),NowTarget.Name)
            if e != nil {
                result.success = false
                return result,e
            }
        } else {
            fmt.Println("There are no files to build.",loaddir)
        }
    } else if NowTarget.Type == "convert" {
        if len(cvfiles) > 0 {
            create_convert(info,loaddir,cvfiles,NowTarget.Name)
        } else {
            fmt.Println("There are no files to convert.",loaddir)
        }
    } else if NowTarget.Type == "passthrough" {
        result.create_list = append(subdir_create_list,create_list...)
    } else {
        //
        // other...
        //
    }
    if verboseMode == true {
        fmt.Println(pathname+" create list:", len(result.create_list))
        for _,rc := range result.create_list {
            fmt.Println("    ",rc)
        }
    }
    result.success = true
    return result,nil
}

//
// writing rules
//
func output_rules(file *os.File) {
    file.WriteString("builddir = "+outputdir+"\n\n")
    file.WriteString("rule compile\n")
    if target_type == "WIN32" {
        file.WriteString("  command = $compile $options -Fo$out $in\n")
        file.WriteString("  description = Compile: $desc\n")
        file.WriteString("  deps = msvc\n\n")
    } else {
        file.WriteString("  command = $compile $options -o $out $in\n")
        file.WriteString("  description = Compile: $desc\n")
        file.WriteString("  depfile = $depf\n")
        file.WriteString("  deps = gcc\n\n")
    }
    file.WriteString("rule ar\n")
    if useResponse == true {
        if target_type == "WIN32" {
            file.WriteString("  command = $ar $options /out:$out @$out.rsp\n")
        } else {
            file.WriteString("  command = $ar $options $out @$out.rsp\n")
        }
        file.WriteString("  description = Archive: $desc\n")
        file.WriteString("  rspfile = $out.rsp\n")
        file.WriteString("  rspfile_content = $in\n\n")
    } else {
        file.WriteString("  command = $ar $options $out $in\n")
        file.WriteString("  description = Archive: $desc\n\n")
    }
    file.WriteString("rule link\n")
    if useResponse == true {
        if target_type == "WIN32" {
            file.WriteString("  command = $link $options /out:$out @$out.rsp\n")
        } else {
            file.WriteString("  command = $link $options -o $out @$out.rsp\n")
        }
        file.WriteString("  description = Link: $desc\n")
        file.WriteString("  rspfile = $out.rsp\n")
        file.WriteString("  rspfile_content = $in\n\n")
    } else {
        file.WriteString("  command = $link $options -o $out $in\n")
        file.WriteString("  description = Link: $desc\n\n")
    }
    file.WriteString("rule convert\n")
    file.WriteString("  command = $convert $options -o $out $in\n")
    file.WriteString("  description = Convert: $desc\n\n")

    // other compile rules.
    for ext,rule := range other_rule_list {
        file.WriteString("rule compile"+ext[1:]+"\n")
        file.WriteString("  command = "+rule.cmd+"\n")
        file.WriteString("  description = "+rule.title+": $desc\n")
        if rule.need_dep == true {
            file.WriteString("  depfile = $depf\n")
            file.WriteString("  deps = gcc\n\n")
        } else {
            file.WriteString("\n")
        }
    }

    // appended original rules.
    for ar,arv := range append_rules {
        file.WriteString("rule "+ar+"\n")
        file.WriteString("  command = "+arv+"\n")
        file.WriteString("  description = "+ar+": $desc\n\n")
    }

    file.WriteString("build always: phony\n\n")
}


//
// writing ninja
//
func outputNinja() {
    if verboseMode == true {
        fmt.Println("output build.ninja")
    }
    file,err := os.Create("build.ninja")
    if err != nil {
        fmt.Println("gobuild: error:",err.Error())
        os.Exit(1)
    }

    // execute build
    output_rules(file)

    for _,bs := range command_list {
        file.WriteString("build "+bs.outfile+": "+bs.cmdtype)
        for _,f := range bs.infiles {
            file.WriteString(" $\n  "+f)
        }
        for _,dep := range bs.depends {
            depstr := strings.Replace(dep,":","$:",1)
            file.WriteString(" $\n  "+depstr)
        }
        if bs.need_cmd_alias {
            file.WriteString("\n  "+bs.cmdtype+" = "+bs.cmd+"\n")
        } else {
            file.WriteString("\n")
        }
        if bs.depfile != "" {
            file.WriteString("  depf = "+bs.depfile+"\n")
        }
        if len(bs.args) > 0 {
            file.WriteString("  options =")
            for i,o := range bs.args {
                if i & 3 == 3 {
                    file.WriteString(" $\n   ")
                }
                ostr := strings.Replace(o,":","$:",1)
                file.WriteString(" "+ostr)
            }
            file.WriteString("\n")
        }
        file.WriteString("  desc = "+bs.outfile+"\n\n")
    }
    for _,oc := range other_rule_file_list {
        file.WriteString("build "+oc.outfile+": "+oc.rule+" "+oc.infile+"\n")
        file.WriteString("  compiler = "+oc.compiler+"\n")
        if oc.include != "" {
            file.WriteString("  include = "+oc.include+"\n")
        }
        if oc.option != "" {
            file.WriteString("  option = "+oc.option+"\n")
        }
        if oc.depend != "" {
            file.WriteString("  depf = "+oc.depend+"\n")
        }
        file.WriteString("  desc = "+oc.outfile+"\n\n")
    }
}


//
// create vcxproj
//
func outputMSBuild(outdir, projname string) {
  var targets []string;

  for _, command := range command_list {
    if command.cmdtype != "compile" {
      continue
    }

    for _, infile := range command.infiles {
      targets = append(targets, strings.Replace(infile, "$:", ":", 1))
    }
  }

  msbuild.ExportProject(targets, outdir, projname)
}


//
// application interface
//
func main() {

    msbuild  := false
    projdir  := ""
    projname := ""

    flag.BoolVar(&verboseMode,"v",false,"verbose mode")
    flag.BoolVar(&isRelease,"release",false,"release build")
    flag.BoolVar(&isDebug,"debug",true,"debug build")
    flag.BoolVar(&isDevelop,"develop",false,"release build")
    flag.BoolVar(&isProduct,"product",false,"release build")
    flag.StringVar(&target_type,"type","default","build target type")
    flag.StringVar(&target_name,"t","","build target name")
    flag.StringVar(&outputdir,"o","build","build directory")
    flag.BoolVar(&msbuild,"msbuild",false,"Export MSBuild project")
    flag.StringVar(&projdir,"msbuild-dir","./","MSBuild project output directory")
    flag.StringVar(&projname,"msbuild-proj","out","MSBuild project name")
    flag.Parse()

    if isDevelop {
        isDebug = false
    }
    if isRelease {
        isDevelop = false
        isDebug = false
    }
    if isProduct {
        isRelease = false
        isDevelop = false
        isDebug = false
    }
    outputdir_set = false
    useResponse = false
    toplevel = true

    ra := flag.Args()
    if len(ra) > 0 && target_name == "" {
        target_name = ra[0]
    }

    append_rules = map[string] string{}
    command_list = []BuildCommand{}
    other_rule_list = map[string] OtherRule{}
    other_rule_file_list = []OtherRuleFile{}

    if target_name != "" {
        fmt.Println("gobuild: make target: "+target_name)
    }
    build_info := BuildInfo{
        variables : map[string] string{"option_prefix":"-"},
        includes : []string{},
        defines : []string{},
        options : []string{},
        archive_options : []string{},
        convert_options :[]string{},
        link_options :[]string{},
        select_target : target_name,
        target: target_name }
    var r,err = build(build_info,"")
    if r.success == false {
        fmt.Println("gobuild: error:",err.Error())
        os.Exit(1)
    }

    nlen := len(command_list) + len(other_rule_file_list)
    if nlen > 0 {

        outputNinja()

        if msbuild {
          outputMSBuild(projdir, projname)
        }

        fmt.Println("gobuild: done.")
    } else {
        fmt.Println("gobuild: empty")
    }
}
//
//
