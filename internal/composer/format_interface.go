package composer

import (
	"fmt"
	"strings"

	ourtypes "github.com/vlad/ast2llm-go/internal/types"
)

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
