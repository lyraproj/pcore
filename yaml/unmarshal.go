package yaml

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	ym "gopkg.in/yaml.v3"
)

func Unmarshal(c px.Context, data []byte) px.Value {
	var ms ym.Node
	err := ym.Unmarshal([]byte(data), &ms)
	if err != nil {
		var itm interface{}
		err2 := ym.Unmarshal([]byte(data), &itm)
		if err2 != nil {
			panic(px.Error(px.ParseError, issue.H{`language`: `YAML`, `detail`: err.Error()}))
		}
		return wrapValue(c, itm)
	}
	return wrapNode(c, &ms)
}

func wrapNode(c px.Context, n *ym.Node) px.Value {
	switch n.Kind {
	case ym.DocumentNode:
		return wrapNode(c, n.Content[0])
	case ym.SequenceNode:
		ms := n.Content
		es := make([]px.Value, len(ms))
		for i, me := range ms {
			es[i] = wrapNode(c, me)
		}
		return types.WrapValues(es)
	case ym.MappingNode:
		ms := n.Content
		top := len(ms)
		es := make([]*types.HashEntry, top/2)
		for i := 0; i < top; i += 2 {
			es[i/2] = types.WrapHashEntry(wrapNode(c, ms[i]), wrapNode(c, ms[i+1]))
		}
		return types.WrapHash(es)
	case ym.ScalarNode:
		var v interface{}
		err := n.Decode(&v)
		if err != nil {
			panic(err)
		}
		return px.Wrap(c, v)
	}
	return px.Undef
}

func wrapValue(c px.Context, v interface{}) px.Value {
	switch v := v.(type) {
	case *ym.Node:
		return wrapNode(c, v)
	case []interface{}:
		vs := make([]px.Value, len(v))
		for i, y := range v {
			vs[i] = wrapValue(c, y)
		}
		return types.WrapValues(vs)
	default:
		return px.Wrap(c, v)
	}
}
