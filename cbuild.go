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
    "os/exec"
    "os"
    "unsafe"
)

//
// data structures
//
type Target struct {
    Name string
    Type string
}
type StringList struct {
    Type string
    Target string
    Debug []string `yaml:",flow"`
    Release []string `yaml:",flow"`
    List []string `yaml:",flow"`
}
type Variable struct {
    Name string
    Value string
    Type string
    Build string
}
type Build struct {
    Name string
    Command string
    Files []string `yaml:",flow"`
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
    Prebuild []Build `yaml:",flow"`
    Postbuild []Build `yaml:",flow"`
    Source []StringList `yaml:",flow"`
    Convert_List []StringList `yaml:",flow"`
    Subdir []StringList `yaml:",flow"`
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
type BuildCommand struct {
    cmd string
    args []string
    title string
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
    select_target string
    target string
    subdir []string
    create_list []string
}

//
// global variables
//
var (
    isDebug bool
    isRelease bool
    target_type string
    target_name string
    outputdir string

    need_dir_list []string
    command_list []BuildCommand
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
            } else {
                for _,r := range i.Release {
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
func create_archive(info BuildInfo,odir string,create_list []string,target_name string) string {

    arname := odir
    if target_type == "WIN32" {
        arname += target_name + ".lib"
    } else {
        arname += "lib" + target_name + ".a"
    }
    arname = filepath.ToSlash(filepath.Clean(arname))

    alist := []string{arname}
    archiver := info.variables["archiver"]
    alist = append(alist,create_list...)

    t := fmt.Sprintf("Library: %s",arname)
    cmd := BuildCommand{
        cmd : archiver,
        args : append(info.archive_options,alist...),
        title : t }
    command_list = append(command_list,cmd)

    return arname
}

//
// link objects
//
func create_link(info BuildInfo,odir string,create_list []string,target_name string) {
    trname := odir
    if target_type == "WIN32" {
        trname += target_name + ".exe"
    } else {
        trname += target_name
    }
    trname = filepath.ToSlash(filepath.Clean(trname))
 
    linker := info.variables["linker"]

    flist := []string{trname}
    flist = append(flist,info.link_options...)
    flist = append(flist,create_list...)

    t := fmt.Sprintf("Linking: %s",trname)
    cmd := BuildCommand{
        cmd : linker,
        args : append([]string{"-o"},flist...),
        title : t }
    command_list = append(command_list,cmd)
    //fmt.Println("-o " + NowTarget.Name + flist)
}

//
// convert objects
//
func create_convert(info BuildInfo,loaddir string,odir string,create_list []string,target_name string) {
    cvname := odir + target_name
    cvname = filepath.ToSlash(filepath.Clean(cvname))
    converter := info.variables["converter"]

    clist := []string{"-o",cvname}
    for _,f := range create_list {
        clist = append(clist,filepath.ToSlash(filepath.Clean(loaddir+f)))
    }

    t := fmt.Sprintf("Convert: %s",cvname)
    cmd := BuildCommand{
        cmd : converter,
        args : append(info.convert_options,clist...),
        title : t }
    command_list = append(command_list,cmd)
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
        e := MyError { str : my_yaml + ": " + err.Error() }
        result.success = false
        return result,e
    }

    //
    // select target
    //
    var NowTarget Target
    for _,t := range d.Target {
        if info.select_target == "" || t.Name == info.select_target {
            NowTarget = t
            if info.target == "" {
                info.target = t.Name
            }
            break
        }
    }
    if NowTarget.Name == "" {
        e := MyError{ str : "No Target" }
        result.success = false
        return result,e
    }
    info.select_target = ""

    opt_pre := info.variables["option_prefix"]
    //
    // get rules
    //
    for _,v := range d.Variable {
        if v.Type == "" || v.Type == target_type {
            info.variables[v.Name] = v.Value
        }
    }
    for _,i := range getList(d.Include,info.target) {
        if filepath.IsAbs(i) == false {
            i = loaddir + i
        }
        abs, err := filepath.Abs(i)
        if err != nil {
            result.success = false
            return result,err
        }
        info.includes = append(info.includes,opt_pre + "I" + abs)
    }
    for _,d := range getList(d.Define,info.target) {
        info.defines = append(info.defines,opt_pre + "D" + d)
    }
    for _,o := range getList(d.Option,info.target) {
        info.options = append(info.options,opt_pre + o)
    }
    for _,a := range getList(d.Archive_Option,info.target) {
        info.archive_options = append(info.archive_options,a)
    }
    for _,c := range getList(d.Convert_Option,info.target) {
        info.convert_options = append(info.convert_options,c)
    }
    for _,l := range getList(d.Link_Option,info.target) {
        info.link_options = append(info.link_options,opt_pre+l)
    }

    files := getList(d.Source,info.target)
    cvfiles := getList(d.Convert_List,info.target)

    // sub-directories
    subdirs := getList(d.Subdir,info.target)
    for _,s := range subdirs {
        sd := loaddir+s
        var r,e = build(info,sd)
        if r.success == false {
            return r,e
        }
        info.create_list = append(info.create_list,r.create_list...)
    }

    compiler := info.variables["compiler"]

    arg1 := append(append(info.includes,info.defines...),info.options...)
    odir := outputdir + "/" + loaddir
    objdir := outputdir + "/" + loaddir + ".objs/"
    need_dir_list = append(need_dir_list,filepath.Clean(objdir))

    create_list := []string{}
    for _,f := range files {
        sname := filepath.ToSlash(filepath.Clean(loaddir+f))
        oname := filepath.ToSlash(filepath.Clean(objdir+f+".o"))
        create_list = append(create_list,oname)

        t := fmt.Sprintf("Compile: %s",sname)
        cmd := BuildCommand{
            cmd : compiler,
            args : append(arg1,"-o",oname,sname),
            title : t }
        command_list = append(command_list,cmd)
    }

    if NowTarget.Type == "library" {
        // archive
        if len(create_list) > 0 {
            arname := create_archive(info,odir,create_list,NowTarget.Name)
            result.create_list = append(info.create_list,arname)
            //fmt.Println(info.archive_options+arname+alist)
        } else {
            fmt.Println("There are no files to build.")
        }
    } else if NowTarget.Type == "execute" {
        // link program
        if len(create_list) > 0 || len(info.create_list) > 0 {
            create_link(info,odir,append(create_list,info.create_list...),NowTarget.Name)
        } else {
            fmt.Println("There are no files to build.")
        }
    } else if NowTarget.Type == "convert" {
        if len(cvfiles) > 0 {
            create_convert(info,loaddir,odir,cvfiles,NowTarget.Name)
        } else {
            fmt.Println("There are no files to convert.")
        }
    } else {
        //
        // othre...
        //
        result.create_list = append(info.create_list,create_list...)
    }
    result.success = true
    return result,nil
}

func main() {

    flag.BoolVar(&isRelease,"release",false,"release build")
    flag.BoolVar(&isDebug,"debug",true,"debug build")
    flag.StringVar(&target_type,"type","default","build target type")
    flag.StringVar(&target_name,"t","","build target name")
    flag.StringVar(&outputdir,"o","build","build directory")
    flag.Parse()

    outputdir += "/" + target_type + "/"
    if isRelease {
        isDebug = false
        outputdir += "Release"
    } else {
        outputdir += "Debug"
    }

    build_info := BuildInfo{
        variables : map[string] string{"option_prefix":"-"},
        includes : []string{},
        defines : []string{},
        options : []string{},
        archive_options : []string{},
        convert_options :[]string{},
        link_options :[]string{},
        create_list :[]string{},
        select_target : target_name,
        target: target_name }
    var r,err = build(build_info,"")
    if r.success == false {
        fmt.Println("Error:",err.Error())
        os.Exit(1)
    }

    // setup directories
    for _,nd := range need_dir_list {
        os.MkdirAll(nd,os.ModePerm)
    }

    // execute build
    nlen := len(command_list)
    if nlen > 0 {
        for i,bs := range command_list {
            t := fmt.Sprintf("[%d/%d] %s",i+1,nlen,bs.title)
            fmt.Println(t)
            //fmt.Println(bs.args)
            c,_ := exec.Command(bs.cmd,bs.args...).CombinedOutput()
            msg := *(*string)(unsafe.Pointer(&c))
            if msg != "" {
                fmt.Println(msg)
            }
        }
    }
}
//
//
