//
//
//
package main

import (
    "fmt"
    "flag"
    "os"
    "bufio"
)

func main() {
    var output_name string

    flag.StringVar(&output_name,"o","","target name")
    flag.Parse()

    args := flag.Args()
    if len(args) == 0 {
        fmt.Println("no input file.")
        os.Exit(1)
    }

    inputname := args[0]
    if output_name == "" {
        output_name = inputname+".cpp"
    }

    fp,err := os.Open(inputname)
    if err != nil {
        fmt.Println(err.Error())
        os.Exit(1)
    }
    ofp,err := os.Create(output_name)
    if err != nil {
        fmt.Println(err.Error())
        os.Exit(1)
    }

    scanner := bufio.NewScanner(fp)
    ofp.WriteString("#include <stdio.h>\n#include <string.h>\nvoid\nputHello()\n{\n")
    for scanner.Scan() {
        str := scanner.Text()
        if str != "" {
            s := fmt.Sprintf("  printf(\"%s\\n\");\n",str)
            ofp.WriteString(s)
        }
    }
    ofp.WriteString("}\n//")
    ofp.Close()

    fp.Close()
}

//
