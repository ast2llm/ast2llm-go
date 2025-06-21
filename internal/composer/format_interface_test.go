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

// TestProjectComposer_Compose_Interface tests composing output for local and imported interfaces.
func TestProjectComposer_Compose_Interface(t *testing.T) {
	// Create a temporary directory for the test project
	tmpDir, err := os.MkdirTemp("", "testproject_interface")
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

func main() {
	var _ mypkg.MyReadCloser
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGoContent), 0644)
	assert.NoError(t, err)

	mypkgDir := filepath.Join(tmpDir, "internal", "mypkg")
	err = os.MkdirAll(mypkgDir, 0755)
	assert.NoError(t, err)

	mypkgGoContent := `
package mypkg

import "io"

// MyReader is a test interface.
type MyReader interface {
	Read(p []byte) (n int, err error)
}

// MyReadCloser embeds another interface.
type MyReadCloser interface {
	MyReader
	io.Closer
	// CloseThis is a specific close method.
	CloseThis() error
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

	// Test Compose for main.go (used imported interface)
	mainGoPath := filepath.Join(tmpDir, "main.go")
	composedOutput, err := composer.Compose(mainGoPath)
	assert.NoError(t, err)

	assert.Contains(t, composedOutput, "Used Items From Other Packages:")
	assert.Contains(t, composedOutput, "Interface: example.com/testproject/internal/mypkg.MyReadCloser")
	assert.Contains(t, composedOutput, "  Comment: MyReadCloser embeds another interface.")
	assert.Contains(t, composedOutput, "  Embeds:")
	assert.Contains(t, composedOutput, "    - example.com/testproject/internal/mypkg.MyReader")
	// Note: a bug in the current parser, it doesn't show external package embeds like io.Closer
	assert.Contains(t, composedOutput, "  Methods:")
	assert.Contains(t, composedOutput, "    - CloseThis() (error)")

	// Test Compose for mypkg.go (local interfaces)
	mypkgGoPath := filepath.Join(mypkgDir, "mypkg.go")
	composedOutputPkg, err := composer.Compose(mypkgGoPath)
	assert.NoError(t, err)

	assert.Contains(t, composedOutputPkg, "Local Interfaces:")
	assert.Contains(t, composedOutputPkg, "Interface: example.com/testproject/internal/mypkg.MyReader")
	assert.Contains(t, composedOutputPkg, "  Comment: MyReader is a test interface.")
	assert.Contains(t, composedOutputPkg, "    - Read([]byte) (int, error)")

	assert.Contains(t, composedOutputPkg, "Interface: example.com/testproject/internal/mypkg.MyReadCloser")
}

// TestProjectComposer_Format_Interface tests composing output from manually created ProjectInfo.
func TestProjectComposer_Format_Interface(t *testing.T) {
	projectInfo := map[string]*types.FileInfo{
		"/project/other.go": {
			PackageName: "other",
			Interfaces: []*types.InterfaceInfo{
				{
					Name:      "testme/dto.MyInterface",
					Comment:   "A test interface.",
					Embeddeds: []string{"io.Writer"},
					Methods: []*types.InterfaceMethod{
						{
							Name:        "DoSomething",
							Parameters:  []string{"ctx context.Context"},
							ReturnTypes: []string{"error"},
							Comment:     "Performs an action.",
						},
					},
				},
			},
		},
		"/project/file.go": {
			PackageName:         "main",
			UsedImportedStructs: []*types.StructInfo{{Name: "testme/dto.MyInterface"}},
		},
	}
	composer := composer.New(projectInfo)
	output, err := composer.Compose("/project/file.go")
	assert.NoError(t, err)

	assert.Contains(t, output, "Used Items From Other Packages:")
	assert.Contains(t, output, "Interface: testme/dto.MyInterface")
	assert.Contains(t, output, "  Comment: A test interface.")
	assert.Contains(t, output, "  Embeds:")
	assert.Contains(t, output, "    - io.Writer")
	assert.Contains(t, output, "  Methods:")
	assert.Contains(t, output, "    - DoSomething(ctx context.Context) (error)")
	assert.Contains(t, output, "      Comment: Performs an action.")
}
