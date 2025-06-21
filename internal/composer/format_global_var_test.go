package composer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vlad/ast2llm-go/internal/composer"
	"github.com/vlad/ast2llm-go/internal/parser"
	"github.com/vlad/ast2llm-go/internal/types"
)

func TestProjectComposer_Compose_GlobalVars(t *testing.T) {
	// Create a temporary directory for the test project
	tmpDir, err := os.MkdirTemp("", "testproject_globals")
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
	"example.com/testproject/internal/mypkg"
)

var _, _ = mypkg.MyPkgConstant, mypkg.MyPkgVariable

func main() {
	
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGoContent), 0644)
	assert.NoError(t, err)

	mypkgDir := filepath.Join(tmpDir, "internal", "mypkg")
	err = os.MkdirAll(mypkgDir, 0755)
	assert.NoError(t, err)

	mypkgGoContent := `
package mypkg

// MyPkgConstant is an awesome constant.
const MyPkgConstant = 42

// MyPkgVariable is an awesome variable.
var MyPkgVariable = "foo"
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

	assert.Contains(t, composedOutput, "Used Items From Other Packages:")
	assert.Contains(t, composedOutput, `  Const: example.com/testproject/internal/mypkg.MyPkgConstant untyped int = 42
    Comment: MyPkgConstant is an awesome constant.`)
	assert.Contains(t, composedOutput, `  Var: example.com/testproject/internal/mypkg.MyPkgVariable string = "foo"
    Comment: MyPkgVariable is an awesome variable.`)

	// Test Compose for mypkg.go to check local vars
	mypkgGoPath := filepath.Join(mypkgDir, "mypkg.go")
	composedOutputPkg, err := composer.Compose(mypkgGoPath)
	assert.NoError(t, err)

	assert.Contains(t, composedOutputPkg, "Global Variables/Constants:")
	assert.Contains(t, composedOutputPkg, `  Const: MyPkgConstant untyped int = 42
    Comment: MyPkgConstant is an awesome constant.`)
	assert.Contains(t, composedOutputPkg, `  Var: MyPkgVariable string = "foo"
    Comment: MyPkgVariable is an awesome variable.`)
}

func TestProjectComposer_Format_GlobalVars(t *testing.T) {
	projectInfo := map[string]*types.FileInfo{
		"/project/other.go": {
			PackageName: "other",
			Imports:     []string{},
			Functions:   []*types.FunctionInfo{},
			GlobalVars: []*types.GlobalVarInfo{
				{
					Name:    "testme/dto.MyGlobalVariable",
					Type:    "time.Duration",
					Value:   "5 * time.Second",
					IsConst: false,
				},
			},
			Structs:             []*types.StructInfo{},
			Interfaces:          []*types.InterfaceInfo{},
			UsedImportedStructs: []*types.StructInfo{},
		},
		"/project/file.go": {
			PackageName: "main",
			Imports:     []string{},
			Functions:   []*types.FunctionInfo{},
			Structs:     []*types.StructInfo{},
			Interfaces:  []*types.InterfaceInfo{},
			UsedImportedStructs: []*types.StructInfo{
				{Name: "testme/dto.MyGlobalVariable"},
			},
		},
	}
	composer := composer.New(projectInfo)
	output, err := composer.Compose("/project/file.go")
	assert.NoError(t, err)
	assert.Contains(t, output, "Used Items From Other Packages:\n- testme/dto.MyGlobalVariable\n")
}
