package types

// FileInfo represents the parsed information about a Go file
type FileInfo struct {
	PackageName            string           // Name of the package
	Imports                []string         // List of imported packages
	Functions              []*FunctionInfo  // List of functions with details
	Structs                []*StructInfo    // List of struct names with their comments, fields, and methods
	Interfaces             []*InterfaceInfo // List of interface names with their comments, methods, and embeddeds
	GlobalVars             []*GlobalVarInfo // List of global variables and constants
	UsedImportedStructs    []*StructInfo    // List of imported struct names used in the file, with fields and methods
	UsedImportedFunctions  []*FunctionInfo  // List of imported function names used in the file, with signature and comment
	UsedImportedGlobalVars []*GlobalVarInfo // List of imported global variables and constants
}

// NewFileInfo creates a new FileInfo instance
func NewFileInfo() *FileInfo {
	return &FileInfo{
		Imports:                make([]string, 0),
		Functions:              make([]*FunctionInfo, 0),
		Structs:                make([]*StructInfo, 0),
		Interfaces:             make([]*InterfaceInfo, 0),
		GlobalVars:             make([]*GlobalVarInfo, 0),
		UsedImportedStructs:    make([]*StructInfo, 0),
		UsedImportedFunctions:  make([]*FunctionInfo, 0),
		UsedImportedGlobalVars: make([]*GlobalVarInfo, 0),
	}
}

// StructField represents a field within a struct
type StructField struct {
	Name string // Field name
	Type string // Field type
}

// StructMethod represents a method associated with a struct
type StructMethod struct {
	Name        string   // Method name
	Comment     string   // Method comment
	Parameters  []string // List of parameter types
	ReturnTypes []string // List of return types
}

// StructInfo represents detailed information about a struct
type StructInfo struct {
	Name    string          // Struct name
	Comment string          // Struct comment
	Fields  []*StructField  // List of fields
	Methods []*StructMethod // List of methods
}

// NewStructInfo creates a new StructInfo instance
func NewStructInfo() *StructInfo {
	return &StructInfo{
		Fields:  make([]*StructField, 0),
		Methods: make([]*StructMethod, 0),
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

// InterfaceMethod represents a method within an interface
type InterfaceMethod struct {
	Name        string   // Method name
	Comment     string   // Method comment
	Parameters  []string // List of parameter types
	ReturnTypes []string // List of return types
}

// InterfaceInfo represents detailed information about an interface
type InterfaceInfo struct {
	Name      string             // Interface name (fully qualified)
	Comment   string             // Interface comment
	Methods   []*InterfaceMethod // List of methods
	Embeddeds []string           // Names of embedded interfaces
}

// NewInterfaceInfo creates a new InterfaceInfo instance
func NewInterfaceInfo() *InterfaceInfo {
	return &InterfaceInfo{
		Methods:   make([]*InterfaceMethod, 0),
		Embeddeds: make([]string, 0),
	}
}

// GlobalVarInfo represents a global variable or constant.
type GlobalVarInfo struct {
	Name    string // Variable name
	Comment string // Associated comment
	Type    string // Variable type
	Value   string // Value, if it's a constant or has a simple literal value
	IsConst bool   // True if it's a constant
}

// FunctionInfo represents detailed information about a function
type FunctionInfo struct {
	Name    string   // Function name (fully qualified)
	Comment string   // Function comment
	Params  []string // List of parameter types (with names if possible)
	Returns []string // List of return types
}
