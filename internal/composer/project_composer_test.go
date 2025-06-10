package composer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vlad/ast2llm-go/internal/composer"
	"github.com/vlad/ast2llm-go/internal/parser"
)

func TestProjectComposer_Compose(t *testing.T) {
	// Create a temporary directory for the test project
	tmpDir, err := os.MkdirTemp("", "testproject")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a dummy go.mod file
	goModContent := `
module example.com/testproject

go 1.22
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
	assert.NoError(t, err)

	// Create dummy files
	mainGoContent := `
package main

import (
	"fmt"
	"log"
	"example.com/testproject/internal/mypkg"
)

// MyStruct is a sample struct.
type MyStruct struct {
	Field1 string // Field1 is a string field.
	Field2 int    // Field2 is an integer field.
}

// MyMethod is a sample method for MyStruct.
func (m *MyStruct) MyMethod(param1 string) (string, error) {
	fmt.Println("Hello from MyMethod")
	return "", nil
}

func main() {
	fmt.Println("Hello, World!")
	var s MyStruct
	s.MyMethod("test")
	_ = mypkg.MyPkgStruct{}
	log.Printf("log test")
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGoContent), 0644)
	assert.NoError(t, err)

	mypkgDir := filepath.Join(tmpDir, "internal", "mypkg")
	err = os.MkdirAll(mypkgDir, 0755)
	assert.NoError(t, err)

	mypkgGoContent := `
package mypkg

// MyPkgStruct is a struct in mypkg.
type MyPkgStruct struct {
	ID   string
	Name string
}

// NewMyPkgStruct creates a new MyPkgStruct.
func NewMyPkgStruct(id, name string) *MyPkgStruct {
	return &MyPkgStruct{ID: id, Name: name}
}
`
	err = os.WriteFile(filepath.Join(mypkgDir, "mypkg.go"), []byte(mypkgGoContent), 0644)
	assert.NoError(t, err)

	// Parse the project
	p := parser.New()
	projectInfo, err := p.ParseProject(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, projectInfo)

	// Create a ProjectComposer
	composer := composer.New(projectInfo)

	// Test Compose for main.go
	mainGoPath := filepath.Join(tmpDir, "main.go")
	composedOutput, err := composer.Compose(mainGoPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, composedOutput)

	// Assertions for main.go content
	assert.Contains(t, composedOutput, "--- File: "+mainGoPath+" ---")
	assert.Contains(t, composedOutput, "Package: main")
	assert.Contains(t, composedOutput, "Imports:\n- fmt\n- log\n- example.com/testproject/internal/mypkg")
	assert.Contains(t, composedOutput, "Functions:\n- main()")
	assert.Contains(t, composedOutput, "Local Structs:\n  Struct: example.com/testproject.MyStruct\n    Comment: MyStruct is a sample struct.\n    Fields:\n      - Field1 string\n      - Field2 int\n    Methods:\n      - MyMethod(string) (string, error)\n        Comment: MyMethod is a sample method for MyStruct.")

	// For imported structs, we now expect full details if they are from the project.
	assert.Contains(t, composedOutput, "Used Imported Structs (from this project, if available):\n  Struct: example.com/testproject/internal/mypkg.MyPkgStruct\n    Comment: MyPkgStruct is a struct in mypkg.\n    Fields:\n      - ID string\n      - Name string")

	// Test Compose for mypkg.go
	mypkgGoPath := filepath.Join(mypkgDir, "mypkg.go")
	composedOutputPkg, err := composer.Compose(mypkgGoPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, composedOutputPkg)

	// Assertions for mypkg.go content
	assert.Contains(t, composedOutputPkg, "--- File: "+mypkgGoPath+" ---")
	assert.Contains(t, composedOutputPkg, "Package: mypkg")
	assert.Contains(t, composedOutputPkg, "Local Structs:\n  Struct: example.com/testproject/internal/mypkg.MyPkgStruct\n    Comment: MyPkgStruct is a struct in mypkg.\n    Fields:\n      - ID string\n      - Name string")
	assert.Contains(t, composedOutputPkg, "Functions:\n- NewMyPkgStruct()")
	// No used imported structs in mypkg.go test case

	// Test for non-existent file
	_, err = composer.Compose(filepath.Join(tmpDir, "nonexistent.go"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file info not found")
}
