package composer

import (
	"fmt"
	"strings"

	ourtypes "github.com/vlad/ast2llm-go/internal/types"
)

// FormatFunction formats a FunctionInfo into the StringBuilder.
func (p *ProjectComposer) FormatFunction(builder *strings.Builder, fn *ourtypes.FunctionInfo, indent string) {
	builder.WriteString(fmt.Sprintf("%sFunction: %s\n", indent, fn.Name))
	if fn.Comment != "" {
		builder.WriteString(fmt.Sprintf("%s  Comment: %s\n", indent, fn.Comment))
	}
	builder.WriteString(fmt.Sprintf("%s  Signature: (%s)", indent, strings.Join(fn.Params, ", ")))
	if len(fn.Returns) > 0 {
		builder.WriteString(fmt.Sprintf(" -> (%s)", strings.Join(fn.Returns, ", ")))
	}
	builder.WriteString("\n")
}
