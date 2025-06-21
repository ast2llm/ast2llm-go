package composer

import (
	"fmt"
	"strings"

	ourtypes "github.com/vlad/ast2llm-go/internal/types"
)

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
