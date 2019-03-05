package types

import (
	"io"

	"github.com/lyraproj/pcore/eval"
)

type (
	DefaultType struct{}

	// DefaultValue is an empty struct because both type and value are known
	DefaultValue struct{}
)

var defaultTypeDefault = &DefaultType{}

var DefaultMetaType eval.ObjectType

func init() {
	DefaultMetaType = newObjectType(`Pcore::DefaultType`, `Pcore::AnyType{}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return DefaultDefaultType()
	})
}

func DefaultDefaultType() *DefaultType {
	return defaultTypeDefault
}

func (t *DefaultType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *DefaultType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*DefaultType)
	return ok
}

func (t *DefaultType) IsAssignable(o eval.Type, g eval.Guard) bool {
	return o == defaultTypeDefault
}

func (t *DefaultType) IsInstance(o eval.Value, g eval.Guard) bool {
	_, ok := o.(*DefaultValue)
	return ok
}

func (t *DefaultType) MetaType() eval.ObjectType {
	return DefaultMetaType
}

func (t *DefaultType) Name() string {
	return `Default`
}

func (t *DefaultType) CanSerializeAsString() bool {
	return true
}

func (t *DefaultType) SerializationString() string {
	return t.String()
}

func (t *DefaultType) String() string {
	return eval.ToString2(t, None)
}

func (t *DefaultType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *DefaultType) PType() eval.Type {
	return &TypeType{t}
}

func WrapDefault() *DefaultValue {
	return &DefaultValue{}
}

func (dv *DefaultValue) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*DefaultValue)
	return ok
}

func (dv *DefaultValue) ToKey() eval.HashKey {
	return eval.HashKey([]byte{1, HkDefault})
}

func (dv *DefaultValue) String() string {
	return `default`
}

func (dv *DefaultValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), dv.PType())
	switch f.FormatChar() {
	case 'd', 's', 'p':
		f.ApplyStringFlags(b, `default`, f.IsAlt())
	case 'D':
		f.ApplyStringFlags(b, `Default`, f.IsAlt())
	default:
		panic(s.UnsupportedFormat(dv.PType(), `dDsp`, f))
	}
}

func (dv *DefaultValue) PType() eval.Type {
	return DefaultDefaultType()
}
