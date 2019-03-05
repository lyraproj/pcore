package types

import (
	"github.com/lyraproj/pcore/eval"
)

var emptyArray = WrapValues([]eval.Value{})
var emptyMap = WrapHash([]*HashEntry{})
var emptyString = stringValue(``)
var undef = WrapUndef()

func init() {
	eval.EmptyArray = emptyArray
	eval.EmptyMap = emptyMap
	eval.EmptyString = emptyString
	eval.Undef = undef
}
