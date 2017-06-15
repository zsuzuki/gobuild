package main

import (
	"path/filepath"
	"os"
	"gopkg.in/myesui/uuid.v1"
)

type TransientOutputPath struct {
	Output     string
	TempOutput string
	done       bool
}

func NewTransientOutput(path string) *TransientOutputPath {
	result := new(TransientOutputPath)
	result.Output = filepath.Clean(path)
	d := filepath.Dir(result.Output)
	id := uuid.NewV4 ()
	result.TempOutput = filepath.Join (d, "gb-" + id.String() + ".tmp")
	result.done = false
	return result
}

func (t *TransientOutputPath) Commit() error {
	if ! t.done {
		t.done = true
		return os.Rename(t.TempOutput, t.Output)
	}
	return nil
}

func (t *TransientOutputPath) Abort() error {
	if ! t.done {
		return os.Remove(t.TempOutput)
	}
	return nil
}

func (t *TransientOutputPath) Done () bool {
	return t.done
}
