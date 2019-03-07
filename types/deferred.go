package types

import (
	"io"

	"github.com/lyraproj/pcore/errors"
	"github.com/lyraproj/pcore/px"
)

var DeferredMetaType px.ObjectType

var DeferredResolve = func(d Deferred, c px.Context) px.Value {
	fn := d.Name()
	args := d.Arguments().AppendTo(make([]px.Value, 0, d.Arguments().Len()))
	for i, a := range args {
		args[i] = ResolveDeferred(c, a)
	}
	return px.Call(c, fn, args, nil)
}

func init() {
	DeferredMetaType = newObjectType(`Deferred`, `{
    attributes => {
      # Fully qualified name of the function
      name  => { type => Pattern[/\A[$]?[a-z][0-9A-Za-z_]*(?:::[a-z][0-9A-Za-z_]*)*\z/] },
      arguments => { type => Optional[Array[Any]], value => undef},
    }}`,
		func(ctx px.Context, args []px.Value) px.Value {
			return newDeferred2(args...)
		},
		func(ctx px.Context, args []px.Value) px.Value {
			return newDeferredFromHash(args[0].(*HashValue))
		})
}

type Deferred interface {
	px.Value

	Name() string

	Arguments() *ArrayValue

	Resolve(c px.Context) px.Value
}

type deferred struct {
	name      string
	arguments *ArrayValue
}

func NewDeferred(name string, arguments ...px.Value) *deferred {
	return &deferred{name, WrapValues(arguments)}
}

func newDeferred2(args ...px.Value) *deferred {
	argc := len(args)
	if argc < 1 || argc > 2 {
		panic(errors.NewIllegalArgumentCount(`deferred[]`, `1 - 2`, argc))
	}
	if name, ok := args[0].(stringValue); ok {
		if argc == 1 {
			return &deferred{string(name), emptyArray}
		}
		if as, ok := args[1].(*ArrayValue); ok {
			return &deferred{string(name), as}
		}
		panic(NewIllegalArgumentType(`deferred[]`, 1, `Array`, args[1]))
	}
	panic(NewIllegalArgumentType(`deferred[]`, 0, `String`, args[0]))
}

func newDeferredFromHash(hash *HashValue) *deferred {
	name := hash.Get5(`name`, px.EmptyString).String()
	arguments := hash.Get5(`arguments`, px.EmptyArray).(*ArrayValue)
	return &deferred{name, arguments}
}

func (e *deferred) Name() string {
	return e.name
}

func (e *deferred) Arguments() *ArrayValue {
	return e.arguments
}

func (e *deferred) String() string {
	return px.ToString(e)
}

func (e *deferred) Equals(other interface{}, guard px.Guard) bool {
	if o, ok := other.(*deferred); ok {
		return e.name == o.name &&
			px.GuardedEquals(e.arguments, o.arguments, guard)
	}
	return false
}

func (e *deferred) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	ObjectToString(e, s, b, g)
}

func (e *deferred) PType() px.Type {
	return DeferredMetaType
}

func (e *deferred) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `name`:
		return stringValue(e.name), true
	case `arguments`:
		return e.arguments, true
	}
	return nil, false
}

func (e *deferred) InitHash() px.OrderedMap {
	return WrapHash([]*HashEntry{WrapHashEntry2(`name`, stringValue(e.name)), WrapHashEntry2(`arguments`, e.arguments)})
}

func (e *deferred) Resolve(c px.Context) px.Value {
	return DeferredResolve(e, c)
}

// ResolveDeferred will resolve all occurrences of a DeferredValue in its
// given argument. Array and Hash arguments will be resolved recursively.
func ResolveDeferred(c px.Context, a px.Value) px.Value {
	switch a := a.(type) {
	case Deferred:
		return a.Resolve(c)
	case *ArrayValue:
		return a.Map(func(v px.Value) px.Value {
			return ResolveDeferred(c, v)
		})
	case *HashValue:
		return a.MapEntries(func(v px.MapEntry) px.MapEntry {
			return WrapHashEntry(ResolveDeferred(c, v.Key()), ResolveDeferred(c, v.Value()))
		})
	default:
		return a
	}
}
