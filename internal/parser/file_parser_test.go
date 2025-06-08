package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vlad/ast2llm-go/internal/types"
)

func TestParseFile(t *testing.T) {
	t.Parallel()

	p := New()

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "valid file",
			input:    "package main; func foo() {}",
			expected: "main",
			wantErr:  false,
		},
		{
			name:    "invalid syntax",
			input:   "package main; func foo() {",
			wantErr: true,
		},
		{
			name:     "empty file",
			input:    "package main",
			expected: "main",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := p.Parse("test.go", []byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, file.Name.Name)
		})
	}
}

func TestExtractDeps(t *testing.T) {
	t.Parallel()

	p := New()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "file with imports and function calls",
			input: `package main

import (
	"fmt"
	"net/http"
)

func foo() {
	fmt.Println("test")
	http.Get("http://example.com")
}`,
			expected: []string{"fmt", "net/http"},
		},
		{
			name: "file with local function calls",
			input: `package main

func localFunc() {}
func main() {
	localFunc()
}`,
			expected: []string{"localFunc"},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := p.Parse("test.go", []byte(tt.input))
			assert.NoError(t, err)

			deps := p.ExtractDeps(file)
			if tt.name == "file with imports and function calls" {
				assert.ElementsMatch(t, []string{"fmt", "net/http", "http"}, deps)
			} else {
				assert.ElementsMatch(t, tt.expected, deps)
			}
		})
	}
}

func TestExtractExportedFunctions(t *testing.T) {
	t.Parallel()

	p := New()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "file with exported and unexported functions",
			input: `package main

func ExportedFunc() {}
func unexportedFunc() {}`,
			expected: []string{"ExportedFunc"},
		},
		{
			name: "file with no exported functions",
			input: `package main

func localFunc() {}`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			file, err := p.Parse("test.go", []byte(tt.input))
			assert.NoError(t, err)

			functions := p.ExtractExportedFunctions(file)
			assert.ElementsMatch(t, tt.expected, functions)
		})
	}
}

func TestBuildGraph(t *testing.T) {
	t.Parallel()

	p := New()

	// Create temporary project structure
	tmpDir := t.TempDir()

	// Create package a
	aDir := filepath.Join(tmpDir, "a")
	err := os.MkdirAll(aDir, 0755)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(aDir, "a.go"), []byte(`package a

func A() {}`), 0644)
	assert.NoError(t, err)

	// Create package b that depends on a
	bDir := filepath.Join(tmpDir, "b")
	err = os.MkdirAll(bDir, 0755)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(bDir, "b.go"), []byte(`package b

import "test/a"

func B() {
	a.A()
}`), 0644)
	assert.NoError(t, err)

	// Initialize Go module
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module test

go 1.21`), 0644)
	assert.NoError(t, err)

	// Build graph
	graph, err := p.BuildGraph(tmpDir)
	assert.NoError(t, err)

	// Verify graph structure
	assert.Contains(t, graph.Nodes, "test/a")
	assert.Contains(t, graph.Nodes, "test/b")

	// Check dependencies
	assert.Contains(t, graph.Nodes["test/b"].DependsOn, "test/a")

	// Check functions
	assert.Contains(t, graph.Nodes["test/a"].Functions, "A")
	assert.Contains(t, graph.Nodes["test/b"].Functions, "B")
}

func TestExtractStructsWithComments(t *testing.T) {
	t.Parallel()

	p := New()

	tests := []struct {
		name     string
		input    string
		expected []*types.StructInfo
	}{
		{
			name: "struct with doc comment, fields, and methods",
			input: `package main

import (
	"fmt"
)

// MyStruct represents a sample structure.
type MyStruct struct {
	Field1 string
	Count  int
}

// Greet says hello.
func (m *MyStruct) Greet() string {
	return fmt.Sprintf("Hello, %s!", m.Field1)
}

// GetCount returns the count.
func (m MyStruct) GetCount(multiplier int) (int, error) {
	return m.Count * multiplier, nil
}
`,
			expected: []*types.StructInfo{
				{
					Name:    "MyStruct",
					Comment: "MyStruct represents a sample structure.",
					Fields: []*types.StructField{
						{Name: "Field1", Type: "string"},
						{Name: "Count", Type: "int"},
					},
					Methods: []*types.StructMethod{
						{Name: "Greet", Comment: "Greet says hello.", Parameters: []string{}, ReturnTypes: []string{"string"}},
						{Name: "GetCount", Comment: "GetCount returns the count.", Parameters: []string{"int"}, ReturnTypes: []string{"int", "error"}},
					},
				},
			},
		},
		{
			name: "struct without comment, with fields, no methods",
			input: `package main

type AnotherStruct struct {
	Field2 int
	Name   string
}
`,
			expected: []*types.StructInfo{
				{
					Name:    "AnotherStruct",
					Comment: "",
					Fields: []*types.StructField{
						{Name: "Field2", Type: "int"},
						{Name: "Name", Type: "string"},
					},
					Methods: []*types.StructMethod{},
				},
			},
		},
		{
			name: "multiple structs with varying details",
			input: `package main

// StructOne is the first struct.
type StructOne struct {}

// DoSomething performs an action.
func (s StructOne) DoSomething() {}

type StructTwo struct {
	ID int
}

// GetID returns the ID.
func (s StructTwo) GetID() int { return s.ID }
`,
			expected: []*types.StructInfo{
				{
					Name:    "StructOne",
					Comment: "StructOne is the first struct.",
					Fields:  []*types.StructField{},
					Methods: []*types.StructMethod{
						{Name: "DoSomething", Comment: "DoSomething performs an action.", Parameters: []string{}, ReturnTypes: []string{}},
					},
				},
				{
					Name:    "StructTwo",
					Comment: "",
					Fields: []*types.StructField{
						{Name: "ID", Type: "int"},
					},
					Methods: []*types.StructMethod{
						{Name: "GetID", Comment: "GetID returns the ID.", Parameters: []string{}, ReturnTypes: []string{"int"}},
					},
				},
			},
		},
		{
			name: "no structs",
			input: `package main

func main() {}
`,
			expected: []*types.StructInfo{},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			file, err := p.Parse("test.go", []byte(tt.input))
			assert.NoError(t, err)

			structs := p.ExtractLocalStructInfo(file)
			assert.ElementsMatch(t, tt.expected, structs, fmt.Sprintf("Failed for case %v", tt.name))
		})
	}
}

func TestExtractUsedImportedStructs(t *testing.T) {
	t.Parallel()

	p := New()

	tests := []struct {
		name     string
		input    string
		expected []*types.StructInfo
	}{
		{
			name: "direct import usage",
			input: `package main

import (
	"context"
)

func main() {
	var myCtx context.Context // Variable declaration using imported type
	_ = myCtx
}
`,
			expected: []*types.StructInfo{
				{Name: "context.Context"},
			},
		},
		{
			name: "import usage in slice",
			input: `package main

import (
	"context"
)

func main() {
	var myCtxs []context.Context // Variable declaration using imported type
	_ = myCtxs
}
`,
			expected: []*types.StructInfo{
				{Name: "context.Context"},
			},
		},
		{
			name: "aliased import usage",
			input: `package main

import (
	myctx "context"
)

func main() {
	var anotherCtx myctx.Context // Variable declaration using aliased imported type
	_ = anotherCtx
}
`,
			expected: []*types.StructInfo{
				{Name: "myctx.Context"},
			},
		},
		{
			name: "multiple imported struct usages",
			input: `package main

import (
	"fmt"
	"net/http"
	"io"
)

func main() {
	client := &http.Client{}
	var req *http.Request
	reader := io.Reader(nil)
	fmt.Println(client, req, reader)
}
`,
			expected: []*types.StructInfo{
				{Name: "http.Client"},
				{Name: "http.Request"},
			},
		},
		{
			name: "no imported struct usage",
			input: `package main

func main() {}
`,
			expected: []*types.StructInfo{},
		},
		{
			name: "struct declaration without usage",
			input: `package main

import (
	"context"
)

type MyCustomStruct struct {
	Ctx context.Context
}
`,
			expected: []*types.StructInfo{
				{Name: "context.Context"},
			},
		},
		{
			name: "struct literal instantiation",
			input: `package main

import (
	"sync"
)

func main() {
	var wg sync.WaitGroup
	_ = wg
}
`,
			expected: []*types.StructInfo{
				{Name: "sync.WaitGroup"},
			},
		},
		{
			name: "variable declaration with imported type",
			input: `package main

import (
	"time"
)

func main() {
	var t time.Time
	_ = t
}
`,
			expected: []*types.StructInfo{
				{Name: "time.Time"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			file, err := p.Parse("test.go", []byte(tt.input))
			assert.NoError(t, err)

			usedStructs := p.ExtractUsedImportedStructInfo(file)
			assert.ElementsMatch(t, tt.expected, usedStructs, fmt.Sprintf("Failed for case %v", tt.name))
		})
	}
}
