package yaml

import (
	"time"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	ym "gopkg.in/yaml.v3"
)

type Value struct {
	px.Value
	Line   int
	Column int
}

func (v *Value) ToKey() px.HashKey {
	return px.ToKey(v.Value)
}

func (v *Value) Unwrap() (uv px.Value) {
	switch x := v.Value.(type) {
	case px.OrderedMap:
		uv = x.MapEntries(func(e px.MapEntry) px.MapEntry {
			return types.WrapHashEntry(
				e.Key().(*Value).Unwrap(),
				e.Value().(*Value).Unwrap())
		})
	case *types.Array:
		uv = x.Map(func(e px.Value) px.Value {
			return e.(*Value).Unwrap()
		})
	default:
		uv = v.Value
	}
	return
}

func Unmarshal(c px.Context, data []byte) px.Value {
	var ms ym.Node
	err := ym.Unmarshal([]byte(data), &ms)
	if err != nil {
		panic(px.Error(px.ParseError, issue.H{`language`: `YAML`, `detail`: err.Error()}))
	}
	return wrapNode(c, &ms)
}

func UnmarshalWithPositions(c px.Context, data []byte) *Value {
	var ms ym.Node
	err := ym.Unmarshal([]byte(data), &ms)
	if err != nil {
		panic(px.Error(px.ParseError, issue.H{`language`: `YAML`, `detail`: err.Error()}))
	}
	return wrapNodeWithPosition(c, &ms)
}

func wrapNode(c px.Context, n *ym.Node) (v px.Value) {
	switch n.Kind {
	case ym.DocumentNode:
		v = wrapNode(c, n.Content[0])
	case ym.SequenceNode:
		ms := n.Content
		es := make([]px.Value, len(ms))
		for i, me := range ms {
			es[i] = wrapNode(c, me)
		}
		v = types.WrapValues(es)
	case ym.MappingNode:
		ms := n.Content
		top := len(ms)
		es := make([]*types.HashEntry, top/2)
		for i := 0; i < top; i += 2 {
			es[i/2] = types.WrapHashEntry(wrapNode(c, ms[i]), wrapNode(c, ms[i+1]))
		}
		v = types.WrapHash(es)
	default:
		v = wrapScalar(c, n)
	}
	return
}

func wrapNodeWithPosition(c px.Context, n *ym.Node) (v *Value) {
	switch n.Kind {
	case ym.DocumentNode:
		v = wrapNodeWithPosition(c, n.Content[0])
	case ym.SequenceNode:
		ms := n.Content
		es := make([]px.Value, len(ms))
		for i, me := range ms {
			es[i] = wrapNodeWithPosition(c, me)
		}
		v = &Value{types.WrapValues(es), n.Line, n.Column}
	case ym.MappingNode:
		ms := n.Content
		top := len(ms)
		es := make([]*types.HashEntry, top/2)
		for i := 0; i < top; i += 2 {
			es[i/2] = types.WrapHashEntry(wrapNodeWithPosition(c, ms[i]), wrapNodeWithPosition(c, ms[i+1]))
		}
		v = &Value{types.WrapHash(es), n.Line, n.Column}
	default:
		v = &Value{wrapScalar(c, n), n.Line, n.Column}
	}
	return
}

func wrapScalar(c px.Context, n *ym.Node) px.Value {
	var v px.Value
	switch n.Tag {
	case `!!null`:
		v = px.Undef
	case `!!bool`:
		var x bool
		if err := n.Decode(&x); err != nil {
			panic(err)
		}
		v = types.WrapBoolean(x)
	case `!!int`:
		var x int64
		if err := n.Decode(&x); err != nil {
			panic(err)
		}
		v = types.WrapInteger(x)
	case `!!float`:
		var x float64
		if err := n.Decode(&x); err != nil {
			panic(err)
		}
		v = types.WrapFloat(x)
	case `!!timestamp`:
		var x time.Time
		if err := n.Decode(&x); err != nil {
			panic(err)
		}
		v = types.WrapTimestamp(x)
	case `!!str`:
		v = types.WrapString(n.Value)
	case `!!binary`:
		v = types.BinaryFromString(n.Value, `%b`)
	default:
		var x interface{}
		err := n.Decode(&x)
		if err != nil {
			panic(err)
		}
		v = px.Wrap(c, x)
	}
	return v
}
