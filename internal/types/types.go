package types

// FileInfo represents the parsed information about a Go file
type FileInfo struct {
	PackageName string   // Name of the package
	Imports     []string // List of imported packages
	Functions   []string // List of function names
}

// NewFileInfo creates a new FileInfo instance
func NewFileInfo() *FileInfo {
	return &FileInfo{
		Imports:   make([]string, 0),
		Functions: make([]string, 0),
	}
}
