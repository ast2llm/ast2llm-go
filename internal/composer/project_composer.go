package composer

import (
	"fmt"
	"strings"

	"github.com/vlad/ast2llm-go/internal/parser"
	ourtypes "github.com/vlad/ast2llm-go/internal/types" // Alias ourtypes
)

// ProjectComposer tranform ProjectInfo to friendly representation for LLM
type ProjectComposer struct {
	projectInfo parser.ProjectInfo
}

// New creates a new ProjectComposer instance
func New(projectInfo parser.ProjectInfo) *ProjectComposer {
	return &ProjectComposer{
		projectInfo: projectInfo,
	}
}

// Compose transforms the ProjectInfo into an LLM-friendly description for a given file path.
func (p *ProjectComposer) Compose(filePath string) (string, error) {
	fileInfo, ok := p.projectInfo[filePath]
	if !ok {
		return "", fmt.Errorf("file info not found for path: %s", filePath)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("--- File: %s ---\n", filePath))
	builder.WriteString(fmt.Sprintf("Package: %s\n", fileInfo.PackageName))
	builder.WriteString("\n")

	if len(fileInfo.Imports) > 0 {
		builder.WriteString("Imports:\n")
		for _, imp := range fileInfo.Imports {
			builder.WriteString(fmt.Sprintf("- %s\n", imp))
		}
		builder.WriteString("\n")
	}

	if len(fileInfo.Functions) > 0 {
		builder.WriteString("Functions:\n")
		for _, fn := range fileInfo.Functions {
			builder.WriteString(fmt.Sprintf("- %s()\n", fn))
		}
		builder.WriteString("\n")
	}

	if len(fileInfo.Structs) > 0 {
		builder.WriteString("Local Structs:\n")
		for _, s := range fileInfo.Structs {
			p.formatStruct(&builder, s, "  ")
		}
	}

	if len(fileInfo.UsedImportedStructs) > 0 {
		builder.WriteString("Used Imported Structs (from this project, if available):\n")
		// Create a map to easily look up all local structs by their fully qualified names
		projectStructsMap := make(map[string]*ourtypes.StructInfo)
		for _, info := range p.projectInfo {
			for _, s := range info.Structs {
				projectStructsMap[s.Name] = s
			}
		}

		for _, s := range fileInfo.UsedImportedStructs {
			// s.Name is already the fully qualified name (e.g., "github.com/vlad/ast2llm-go/internal/types.FileInfo")
			if detailedStruct, ok := projectStructsMap[s.Name]; ok {
				// Found a detailed definition within the project
				p.formatStruct(&builder, detailedStruct, "  ")
			} else {
				// External imported struct, or not found within the project's parsed info
				builder.WriteString(fmt.Sprintf("- %s\n", s.Name))
			}
		}
	}

	return builder.String(), nil
}

// formatStruct formats a StructInfo into the StringBuilder.
func (p *ProjectComposer) formatStruct(builder *strings.Builder, s *ourtypes.StructInfo, indent string) {
	builder.WriteString(fmt.Sprintf("%sStruct: %s\n", indent, s.Name))
	if s.Comment != "" {
		builder.WriteString(fmt.Sprintf("%s  Comment: %s\n", indent, s.Comment))
	}

	if len(s.Fields) > 0 {
		builder.WriteString(fmt.Sprintf("%s  Fields:\n", indent))
		for _, f := range s.Fields {
			builder.WriteString(fmt.Sprintf("%s    - %s %s\n", indent, f.Name, f.Type))
		}
	}

	if len(s.Methods) > 0 {
		builder.WriteString(fmt.Sprintf("%s  Methods:\n", indent))
		for _, m := range s.Methods {
			builder.WriteString(fmt.Sprintf("%s    - %s(%s) (%s)\n", indent, m.Name, strings.Join(m.Parameters, ", "), strings.Join(m.ReturnTypes, ", ")))
			if m.Comment != "" {
				builder.WriteString(fmt.Sprintf("%s      Comment: %s\n", indent, m.Comment))
			}
		}
	}
}
