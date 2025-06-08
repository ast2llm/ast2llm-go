package types

// FileInfo represents the parsed information about a Go file
type FileInfo struct {
	PackageName string   // Name of the package
	Imports     []string // List of imported packages
	Functions   []string // List of function names
	Structs     []string // List of struct names with their comments
}

// NewFileInfo creates a new FileInfo instance
func NewFileInfo() *FileInfo {
	return &FileInfo{
		Imports:   make([]string, 0),
		Functions: make([]string, 0),
		Structs:   make([]string, 0),
	}
}

// Node represents a package in the dependency graph
type Node struct {
	PkgPath   string   // Package path
	Functions []string // Exported functions
	DependsOn []string // Imported packages
	Files     []string // Source files in the package
}

// DependencyGraph represents the project's dependency structure
type DependencyGraph struct {
	Nodes map[string]*Node // Key: package path
}

// NewDependencyGraph creates a new DependencyGraph instance
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*Node),
	}
}
