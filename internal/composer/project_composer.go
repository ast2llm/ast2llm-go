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
			p.FormatFunction(&builder, fn, "  ")
		}
		builder.WriteString("\n")
	}

	if len(fileInfo.Structs) > 0 {
		builder.WriteString("Local Structs:\n")
		for _, s := range fileInfo.Structs {
			p.FormatStruct(&builder, s, "  ")
		}
	}

	if len(fileInfo.Interfaces) > 0 {
		builder.WriteString("Local Interfaces:\n")
		for _, iface := range fileInfo.Interfaces {
			p.FormatInterface(&builder, iface, "  ")
		}
	}

	if len(fileInfo.UsedImportedStructs) > 0 || len(fileInfo.UsedImportedFunctions) > 0 {
		builder.WriteString("Used Imported Structs (from this project, if available):\n")
		// Create maps to look up all local structs, interfaces, and functions by their fully qualified names
		projectStructsMap := make(map[string]*ourtypes.StructInfo)
		projectInterfacesMap := make(map[string]*ourtypes.InterfaceInfo)
		projectFunctionsMap := make(map[string]*ourtypes.FunctionInfo)
		for _, info := range p.projectInfo {
			for _, s := range info.Structs {
				projectStructsMap[s.Name] = s
			}
			for _, i := range info.Interfaces {
				projectInterfacesMap[i.Name] = i
			}
			for _, f := range info.Functions {
				projectFunctionsMap[f.Name] = f
			}
		}

		for _, s := range fileInfo.UsedImportedStructs {
			if detailedStruct, ok := projectStructsMap[s.Name]; ok {
				p.FormatStruct(&builder, detailedStruct, "  ")
			} else if detailedIface, ok := projectInterfacesMap[s.Name]; ok {
				p.FormatInterface(&builder, detailedIface, "  ")
			} else if detailedFunc, ok := projectFunctionsMap[s.Name]; ok {
				p.FormatFunction(&builder, detailedFunc, "  ")
			} else {
				builder.WriteString(fmt.Sprintf("- %s\n", s.Name))
			}
		}
		for _, f := range fileInfo.UsedImportedFunctions {
			p.FormatFunction(&builder, f, "  ")
		}
	}

	return builder.String(), nil
}

// FormatStruct formats a StructInfo into the StringBuilder.
func (p *ProjectComposer) FormatStruct(builder *strings.Builder, s *ourtypes.StructInfo, indent string) {
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

// FormatInterface formats an InterfaceInfo into the StringBuilder.
func (p *ProjectComposer) FormatInterface(builder *strings.Builder, iface *ourtypes.InterfaceInfo, indent string) {
	builder.WriteString(fmt.Sprintf("%sInterface: %s\n", indent, iface.Name))
	if iface.Comment != "" {
		builder.WriteString(fmt.Sprintf("%s  Comment: %s\n", indent, iface.Comment))
	}
	if len(iface.Embeddeds) > 0 {
		builder.WriteString(fmt.Sprintf("%s  Embeds:\n", indent))
		for _, emb := range iface.Embeddeds {
			builder.WriteString(fmt.Sprintf("%s    - %s\n", indent, emb))
		}
	}
	if len(iface.Methods) > 0 {
		builder.WriteString(fmt.Sprintf("%s  Methods:\n", indent))
		for _, m := range iface.Methods {
			builder.WriteString(fmt.Sprintf("%s    - %s(%s) (%s)\n", indent, m.Name, strings.Join(m.Parameters, ", "), strings.Join(m.ReturnTypes, ", ")))
			if m.Comment != "" {
				builder.WriteString(fmt.Sprintf("%s      Comment: %s\n", indent, m.Comment))
			}
		}
	}
}
