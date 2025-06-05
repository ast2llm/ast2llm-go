package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
