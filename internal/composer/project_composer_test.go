package composer_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vlad/ast2llm-go/internal/composer"
	"github.com/vlad/ast2llm-go/internal/parser"
	"github.com/vlad/ast2llm-go/internal/types"
)

func TestProjectComposer_Compose_FileNotFound(t *testing.T) {
	projectInfo := parser.ProjectInfo{}
	composer := composer.New(projectInfo)

	_, err := composer.Compose("/path/to/nonexistent.go")
	assert.Error(t, err)
	assert.EqualError(t, err, "file info not found for path: /path/to/nonexistent.go")
}

func TestProjectComposer_Compose_EmptyFileInfo(t *testing.T) {
	filePath := "/path/to/empty.go"
	projectInfo := parser.ProjectInfo{
		filePath: {
			PackageName: "main",
		},
	}
	composer := composer.New(projectInfo)

	output, err := composer.Compose(filePath)
	assert.NoError(t, err)

	expected := `--- File: /path/to/empty.go ---
Package: main

`
	assert.Equal(t, expected, output)
}

func TestProjectComposer_Compose_UnresolvedImport(t *testing.T) {
	filePath := "/project/main.go"
	projectInfo := parser.ProjectInfo{
		filePath: {
			PackageName: "main",
			UsedImportedStructs: []*types.StructInfo{
				{Name: "github.com/some/external/pkg.SomeType"},
			},
		},
		// No info about github.com/some/external/pkg in the project
	}
	composer := composer.New(projectInfo)

	output, err := composer.Compose(filePath)
	assert.NoError(t, err)

	assert.Contains(t, output, "Used Items From Other Packages:")
	assert.Contains(t, output, "- github.com/some/external/pkg.SomeType")
}

func TestProjectComposer_Compose_DeduplicatesUsedItems(t *testing.T) {
	filePath := "/project/main.go"
	projectInfo := parser.ProjectInfo{
		"/project/other.go": {
			PackageName: "other",
			Functions: []*types.FunctionInfo{
				{
					Name:   "example.com/project/other.MyFunction",
					Params: []string{"i int"},
				},
			},
		},
		filePath: {
			PackageName: "main",
			// MyFunction is present in both lists, which could happen due to a parser bug
			UsedImportedStructs: []*types.StructInfo{
				{Name: "example.com/project/other.MyFunction"},
			},
			UsedImportedFunctions: []*types.FunctionInfo{
				{Name: "example.com/project/other.MyFunction", Params: []string{"i int"}},
			},
		},
	}
	composer := composer.New(projectInfo)

	output, err := composer.Compose(filePath)
	assert.NoError(t, err)

	// The function should only be listed once.
	count := strings.Count(output, "Function: example.com/project/other.MyFunction")
	assert.Equal(t, 1, count, "The same used item should not be printed multiple times")
}
