package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// CompileDbItem represents an entry for json compilation database (https://clang.llvm.org/docs/JSONCompilationDatabase.html)
type CompileDbItem struct {
	// The working directory
	Directory string `json:"directory"`
	// The source file
	File string `json:"file"`
	// The output
	Output string `json:"output"`
	// Compilation command
	Arguments []string `json:"arguments"`
}

// CreateCompileDbFile creates compilation-database file.
func CreateCompileDbFile(outPath string, defs []CompileDbItem) error {
	var err error

	outDir := filepath.Dir(outPath)
	tmpOut, err := ioutil.TempFile(outDir, "cdb-")
	if err != nil {
		return errors.Wrapf(err, "failed to create temporal output for \"%s\"", outPath)
	}
	defer (func() {
		tmpOut.Close()
		os.Remove(tmpOut.Name())
	})()
	err = WriteCompileDb(tmpOut, defs)
	if err != nil {
		errors.Wrapf(err, "failed to write definitions")
	}
	tmpOut.Close()
	err = os.Rename(tmpOut.Name(), outPath)
	if err != nil {
		return errors.Wrapf(err, "failed to rename \"%s\" to \"%s\"", tmpOut.Name(), outPath)
	}
	return nil
}

// WriteCompileDb writes definitions to output.
func WriteCompileDb(output io.Writer, defs []CompileDbItem) error {
	var err error
	b, err := json.MarshalIndent(defs, "", "    ")
	if err != nil {
		return errors.Wrapf(err, "failed to marshal definitions")
	}
	cnt, err := output.Write(b)
	if err != nil || cnt != len(b) {
		return errors.Wrapf(err, "failed to write marshaled definitions")
	}
	return nil
}
