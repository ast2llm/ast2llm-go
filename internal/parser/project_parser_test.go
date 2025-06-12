package parser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	ourtypes "github.com/vlad/ast2llm-go/internal/types" // Alias ourtypes
)

func TestProjectParser_ParseProject(t *testing.T) {
	t.Parallel()

	p := New()

	tests := []struct {
		name              string
		projectFiles      map[string]string
		expectedFileInfos map[string]*ourtypes.FileInfo
		wantErr           bool
	}{
		{
			name: "simple project with one file, local struct, and function",
			projectFiles: map[string]string{
				"main.go": `package main

import "fmt"

// MyStruct represents a sample structure.
type MyStruct struct{
	Field1 string
	Count int
}

// Greet says hello.
func (m *MyStruct) Greet() string {
	fmt.Println("Hello")
	return "hello"
}

func main(){
	_ = MyStruct{}
}
`,
			},
			expectedFileInfos: map[string]*ourtypes.FileInfo{
				"/testproject/main.go": {
					PackageName: "main",
					Imports:     []string{"fmt"},
					Functions:   []string{"Greet", "main"},
					Structs: []*ourtypes.StructInfo{
						{
							Name:    "MyStruct",
							Comment: "MyStruct represents a sample structure.",
							Fields: []*ourtypes.StructField{
								{Name: "Field1", Type: "string"},
								{Name: "Count", Type: "int"},
							},
							Methods: []*ourtypes.StructMethod{
								{
									Name:        "Greet",
									Comment:     "Greet says hello.",
									Parameters:  []string{},
									ReturnTypes: []string{"string"},
								},
							},
						},
					},
					UsedImportedStructs: []*ourtypes.StructInfo{},
				},
			},
		},
		{
			name: "project with multiple files and inter-package struct usage",
			projectFiles: map[string]string{
				"pkg1/types.go": `package pkg1

// Data struct
type Data struct {
	Value string
}
`,
				"pkg2/consumer.go": `package pkg2

import (
	"fmt"
	"example.com/testproject/pkg1"
)

// ProcessData processes data.
func ProcessData(d pkg1.Data) {
	fmt.Println(d.Value)
}
`,
			},
			expectedFileInfos: map[string]*ourtypes.FileInfo{
				"/testproject/pkg1/types.go": {
					PackageName: "pkg1",
					Imports:     []string{},
					Functions:   []string{},
					Structs: []*ourtypes.StructInfo{
						{
							Name:    "Data",
							Comment: "Data struct",
							Fields: []*ourtypes.StructField{
								{Name: "Value", Type: "string"},
							},
							Methods: []*ourtypes.StructMethod{},
						},
					},
					UsedImportedStructs: []*ourtypes.StructInfo{},
				},
				"/testproject/pkg2/consumer.go": {
					PackageName: "pkg2",
					Imports:     []string{"fmt", "example.com/testproject/pkg1"},
					Functions:   []string{"ProcessData"},
					Structs:     []*ourtypes.StructInfo{},
					UsedImportedStructs: []*ourtypes.StructInfo{
						{Name: "example.com/testproject/pkg1.Data"},
					},
				},
			},
		},
		{
			name: "empty project",
			projectFiles: map[string]string{
				"empty.go": `package empty`,
			},
			expectedFileInfos: map[string]*ourtypes.FileInfo{
				"/testproject/empty.go": {
					PackageName:         "empty",
					Imports:             []string{},
					Functions:           []string{},
					Structs:             []*ourtypes.StructInfo{},
					UsedImportedStructs: []*ourtypes.StructInfo{},
				},
			},
		},
		{
			name: "struct with anonymous fields and embedded structs",
			projectFiles: map[string]string{
				"main.go": `package main

import "io"

// ReaderWriter struct
type ReaderWriter struct {
	io.Reader
	Writer
	string
}

// Writer interface
type Writer interface {
	Write([]byte) (int, error)
}
`,
			},
			expectedFileInfos: map[string]*ourtypes.FileInfo{
				"/testproject/main.go": {
					PackageName: "main",
					Imports:     []string{"io"},
					Functions:   []string{},
					Structs: []*ourtypes.StructInfo{
						{
							Name:    "ReaderWriter",
							Comment: "ReaderWriter struct",
							Fields: []*ourtypes.StructField{
								{Name: "Reader", Type: "io.Reader"},
								{Name: "Writer", Type: "example.com/testproject.Writer"},
								{Name: "string", Type: "string"},
							},
							Methods: []*ourtypes.StructMethod{},
						},
					},
					UsedImportedStructs: []*ourtypes.StructInfo{
						{Name: "io.Reader"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			projectPath := filepath.Join(tmpDir, "testproject") // Ensure it's a sub-directory
			err := os.MkdirAll(projectPath, 0755)
			assert.NoError(t, err)

			// Create go.mod file with a proper module path
			err = os.WriteFile(filepath.Join(projectPath, "go.mod"), []byte(fmt.Sprintf("module %s\ngo 1.21", "example.com/testproject")), 0644)
			assert.NoError(t, err, "failed to write go.mod")

			// Write project files
			for filePath, content := range tt.projectFiles {
				absPath := filepath.Join(projectPath, filePath)
				err = os.MkdirAll(filepath.Dir(absPath), 0755)
				assert.NoError(t, err)
				err = os.WriteFile(absPath, []byte(content), 0644)
				assert.NoError(t, err)
			}

			// Run go mod tidy to resolve dependencies
			cmd := exec.Command("go", "mod", "tidy")
			cmd.Dir = projectPath
			cmd.Stderr = os.Stderr // Capture stderr for debugging
			cmd.Stdout = os.Stdout // Capture stdout for debugging
			err = cmd.Run()
			assert.NoError(t, err, "go mod tidy failed for project: %s", projectPath)

			fileInfos, err := p.ParseProject(projectPath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Len(t, fileInfos, len(tt.expectedFileInfos))
			for expectedRelativePath, expectedInfo := range tt.expectedFileInfos {
				// Construct the actual absolute path based on the temporary directory
				actualAbsolutePath := filepath.Join(projectPath, strings.TrimPrefix(expectedRelativePath, "/testproject/"))
				actualInfo, ok := fileInfos[actualAbsolutePath]
				assert.True(t, ok, "File %s (expected relative: %s) not found in parsed result", actualAbsolutePath, expectedRelativePath)
				if !ok {
					continue
				}

				assert.Equal(t, expectedInfo.PackageName, actualInfo.PackageName, "Package name mismatch for %s", actualAbsolutePath)

				// Sort imports and functions for consistent comparison
				sort.Strings(expectedInfo.Imports)
				sort.Strings(actualInfo.Imports)
				assert.ElementsMatch(t, expectedInfo.Imports, actualInfo.Imports, "Imports mismatch for %s", actualAbsolutePath)

				sort.Strings(expectedInfo.Functions)
				sort.Strings(actualInfo.Functions)
				assert.ElementsMatch(t, expectedInfo.Functions, actualInfo.Functions, "Functions mismatch for %s", actualAbsolutePath)

				// Compare structs in more detail
				assert.Len(t, actualInfo.Structs, len(expectedInfo.Structs), "Struct count mismatch for %s", actualAbsolutePath)
				for _, expectedStruct := range expectedInfo.Structs {
					// Find the matching actual struct by name
					var actualStruct *ourtypes.StructInfo
					for _, as := range actualInfo.Structs {
						if as.Name == expectedStruct.Name {
							actualStruct = as
							break
						}
					}
					assert.NotNil(t, actualStruct, "Expected struct %s not found in actual structs for %s", expectedStruct.Name, actualAbsolutePath)
					if actualStruct == nil {
						continue
					}

					assert.Equal(t, expectedStruct.Name, actualStruct.Name, "Struct name mismatch for %s in %s", expectedStruct.Name, actualAbsolutePath)
					assert.Equal(t, expectedStruct.Comment, actualStruct.Comment, "Struct comment mismatch for %s in %s", expectedStruct.Name, actualAbsolutePath)

					// Compare fields
					assert.Len(t, actualStruct.Fields, len(expectedStruct.Fields), "Field count mismatch for %s in %s", expectedStruct.Name, actualAbsolutePath)
					for j, expectedField := range expectedStruct.Fields {
						assert.Equal(t, expectedField.Name, actualStruct.Fields[j].Name, "Field name mismatch for %s.%s in %s", expectedStruct.Name, expectedField.Name, actualAbsolutePath)
						assert.Equal(t, expectedField.Type, actualStruct.Fields[j].Type, "Field type mismatch for %s.%s in %s", expectedStruct.Name, expectedField.Name, actualAbsolutePath)
					}

					// Compare methods
					assert.Len(t, actualStruct.Methods, len(expectedStruct.Methods), "Method count mismatch for %s in %s", expectedStruct.Name, actualAbsolutePath)
					for j, expectedMethod := range expectedStruct.Methods {
						assert.Equal(t, expectedMethod.Name, actualStruct.Methods[j].Name, "Method name mismatch for %s.%s in %s", expectedStruct.Name, expectedMethod.Name, actualAbsolutePath)
						assert.Equal(t, expectedMethod.Comment, actualStruct.Methods[j].Comment, "Method comment mismatch for %s.%s in %s", expectedStruct.Name, expectedMethod.Name, actualAbsolutePath)
						assert.ElementsMatch(t, expectedMethod.Parameters, actualStruct.Methods[j].Parameters, "Method parameters mismatch for %s.%s in %s", expectedStruct.Name, expectedMethod.Name, actualAbsolutePath)
						assert.ElementsMatch(t, expectedMethod.ReturnTypes, actualStruct.Methods[j].ReturnTypes, "Method return types mismatch for %s.%s in %s", expectedStruct.Name, expectedMethod.Name, actualAbsolutePath)
					}
				}

				// Compare UsedImportedStructs (only name is populated)
				expectedUsedStructNames := make([]string, 0, len(expectedInfo.UsedImportedStructs))
				for _, s := range expectedInfo.UsedImportedStructs {
					expectedUsedStructNames = append(expectedUsedStructNames, s.Name)
				}

				actualUsedStructNames := make([]string, 0, len(actualInfo.UsedImportedStructs))
				for _, s := range actualInfo.UsedImportedStructs {
					actualUsedStructNames = append(actualUsedStructNames, s.Name)
				}

				sort.Strings(expectedUsedStructNames)
				sort.Strings(actualUsedStructNames)
				assert.Equal(t, expectedUsedStructNames, actualUsedStructNames, "Used imported struct names mismatch for %s", actualAbsolutePath)
			}
		})
	}
}
