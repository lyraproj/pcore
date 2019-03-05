package loader

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/eval"
	"regexp"
	"strings"
)

var aliasPattern = regexp.MustCompile(`(?m:^(?:\s*#.*\n)*\s*type\s+([A-Z][\w]*(?:::[A-Z][\w]*)*)\s*=\s*(.*)$)`)

// InstantiatePuppetType reads the contents a puppet manifest file and parses it using
// the types.Parse() function.
func InstantiatePuppetType(ctx eval.Context, loader ContentProvidingLoader, tn eval.TypedName, sources []string) {
	content := string(loader.GetContent(ctx, sources[0]))
	m := aliasPattern.FindStringSubmatch(content)
	var name string
	if m != nil {
		name = m[1]
		if !strings.EqualFold(tn.Name(), name) {
			panic(eval.Error(eval.WrongDefinition, issue.H{`source`: sources[0], `type`: eval.NsType, `expected`: tn.Name(), `actual`: name}))
		}
		content = m[2]
	} else {
		name = tn.Name()
	}
	eval.AddTypes(ctx, eval.NewNamedType(name, content))
}
