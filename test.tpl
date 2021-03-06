{{/*
Following properties/functions are available:
 - Properties
    - TemplateFile       string     // Template file
    - Platform           string     // Platform identifier ("WIN32", "Mac"...)
    - UseResponse        bool       // Prefer using a response file to pass the lengthy arguments
    - NewlineAsDelimiter bool       // When using a response file, delimit items with '\n' instead of '\x20'
    - GroupArchives      bool       // Groups library items (for symbol resolution)
    - OutputDirectory    string     // The output directory
    - OtherRules         map[string]OtherRule   // extension to rule map
    - AppendRules        map[string]AppendBuild // custom build rules to command map
    - UsePCH             bool       // Prefer using pre-compiled header
    - UseDepsMsvc        bool       // Use MSVC depend format
    - NinjaUpdater       string     // Command for updating *.ninja itself
    - Commands         []*BuildCommand  // List of build commands
    - OtherRuleTargets []OtherRuleFile  // List of targets using custom rules
    - SubNinjas        []string
    - NinjaFile        string       // Name of the output
    - ConfigSources    []string     // Files referenced to build the output
  - Functions
    - join          // join <list> <sep>
    - escape_drive  // Escapes ':'
*/ -}}
# AUTOGENERATED {{if .TemplateFile}}using {{.TemplateFile}}{{end}}
# Rule definitions
builddir = {{.OutputDirectory}}

rule compile
    description = Compiling: $desc
{{- if eq .Platform "WIN32"}}
    command = {{.CompilerLauncher}} $compile $options -Fo$out $in
    {{- if eq .UseDepsMsvc}}
    deps = msvc
    {{- else}}
    deps = gcc
    {{- end}}
{{- else}}
    command = {{.CompilerLauncher}} $compile $options -o $out $in
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
{{- if .UseResponse}}
    description = Linking: $desc
    command = $link $options {{if eq .Platform "WIN32"}}/out:$out{{else}}-o $out{{end}} @$out.rsp
    rspfile = $out.rsp
    rspfile_content = {{if .NewlineAsDelimiter}}$in_newline{{else}}$in{{end}}
{{- else}}
    command = $link $options -o $out {{if .GroupArchives}}-Wl,--start-group $in -Wl,--end-group{{else}}$in{{end}}
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

{{- define "IMPDEPS_"}}
    {{- if .}} | {{join . " "}}{{end}}
{{- end}}
{{/* Render rules */}}

# Commands
build {{.NinjaFile | escape_drive}} : update_ninja_file {{join .ConfigSources " "}}
    desc = {{.NinjaFile}}
{{range $c := .Commands}}
build {{$c.OutFile}} : {{$c.CommandType}} {{join $c.InFiles " "}} {{join $c.Depends " "}} {{template "IMPDEPS_" $c.ImplicitDepends}}
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
# Other targets
{{range $item := .OtherRuleTargets}}
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
