package loader

import (
	"regexp"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

var aliasPattern = regexp.MustCompile(`(?m:^(?:\s*#.*\n)*\s*type\s+([A-Z][\w]*(?:::[A-Z][\w]*)*)\s*=\s*(.*)$)`)

// InstantiatePuppetType reads the contents a puppet manifest file and parses it using
// the types.Parse() function.
func InstantiatePuppetType(ctx px.Context, loader ContentProvidingLoader, tn px.TypedName, sources []string) {
	content := string(loader.GetContent(ctx, sources[0]))
	m := aliasPattern.FindStringSubmatch(content)
	var name string
	if m != nil {
		name = m[1]
		if !strings.EqualFold(tn.Name(), name) {
			panic(px.Error(px.WrongDefinition, issue.H{`source`: sources[0], `type`: px.NsType, `expected`: tn.Name(), `actual`: name}))
		}
		content = m[2]
	} else {
		name = tn.Name()
	}
	px.AddTypes(ctx, px.NewNamedType(name, content))
}
