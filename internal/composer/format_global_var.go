package composer

import (
	"fmt"
	"strings"

	ourtypes "github.com/vlad/ast2llm-go/internal/types"
)

// FormatGlobalVar formats a GlobalVarInfo into the StringBuilder.
func (p *ProjectComposer) FormatGlobalVar(builder *strings.Builder, gv *ourtypes.GlobalVarInfo, indent string) {
	kind := "Var"
	if gv.IsConst {
		kind = "Const"
	}
	builder.WriteString(fmt.Sprintf("%s%s: %s %s", indent, kind, gv.Name, gv.Type))
	if gv.Value != "" {
		builder.WriteString(fmt.Sprintf(" = %s", gv.Value))
	}
	builder.WriteString("\n")

	if gv.Comment != "" {
		builder.WriteString(fmt.Sprintf("%s  Comment: %s\n", indent, gv.Comment))
	}
}
