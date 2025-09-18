package common

type File[T any] struct {
	Name           string
	Content        []byte
	FSSpecificData *T
}

// type FileSystem interface {
// 	ReadFile(path string) (*File, error)
// 	CreateFile(path string, b []byte) (*File, error)
// 	CreateDir(path string) (*File, error)
// 	PrintInfo()
// }
