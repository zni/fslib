package common

import "fmt"

type FSError struct {
	Op   string
	Path string
	Err  error
}

func (e *FSError) Error() string { return fmt.Sprintf("%s %s: %s", e.Op, e.Path, e.Err.Error()) }
func (e *FSError) Unwrap() error { return e.Err }

type FileError struct {
	Op   string
	Path string
	Err  error
}

func (e *FileError) Error() string { return fmt.Sprintf("%s %s: %s", e.Op, e.Path, e.Err.Error()) }
func (e *FileError) Unwrap() error { return e.Err }
