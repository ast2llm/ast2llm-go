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

	if len(fileInfo.GlobalVars) > 0 {
		builder.WriteString("Global Variables/Constants:\n")
		for _, gv := range fileInfo.GlobalVars {
			p.FormatGlobalVar(&builder, gv, "  ")
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

	if len(fileInfo.UsedImportedStructs) > 0 || len(fileInfo.UsedImportedFunctions) > 0 || len(fileInfo.UsedImportedGlobalVars) > 0 {
		builder.WriteString("Used Items From Other Packages:\n")
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

		processedItems := make(map[string]bool)

		for _, s := range fileInfo.UsedImportedStructs {
			if processedItems[s.Name] {
				continue
			}
			if detailedStruct, ok := projectStructsMap[s.Name]; ok {
				p.FormatStruct(&builder, detailedStruct, "  ")
				processedItems[s.Name] = true
			} else if detailedIface, ok := projectInterfacesMap[s.Name]; ok {
				p.FormatInterface(&builder, detailedIface, "  ")
				processedItems[s.Name] = true
			} else if detailedFunc, ok := projectFunctionsMap[s.Name]; ok {
				p.FormatFunction(&builder, detailedFunc, "  ")
				processedItems[s.Name] = true
			} else {
				builder.WriteString(fmt.Sprintf("- %s\n", s.Name))
				processedItems[s.Name] = true
			}
		}
		for _, f := range fileInfo.UsedImportedFunctions {
			if processedItems[f.Name] {
				continue
			}
			p.FormatFunction(&builder, f, "  ")
			processedItems[f.Name] = true
		}
		for _, gv := range fileInfo.UsedImportedGlobalVars {
			if processedItems[gv.Name] {
				continue
			}
			p.FormatGlobalVar(&builder, gv, "  ")
			processedItems[gv.Name] = true
		}
	}

	return builder.String(), nil
}
