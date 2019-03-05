package types_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/types"

	// Ensure that pcore is initialized
	_ "github.com/lyraproj/pcore/pcore"
)

func ExampleUniqueValues() {
	x := types.WrapString(`hello`)
	y := types.WrapInteger(32)
	types.UniqueValues([]eval.Value{x, y})

	z := types.WrapString(`hello`)
	sv := []eval.StringValue{x, z}
	fmt.Println(types.UniqueValues([]eval.Value{sv[0], sv[1]}))
	// Output: [hello]
}

func ExampleNewCallableType() {
	cc := types.NewCallableType(types.NewTupleType([]eval.Type{types.DefaultUnitType()}, types.PositiveIntegerType()), nil, nil)
	fmt.Println(cc)
	// Output: Callable[0, default]
}

func ExampleNewTupleType() {
	tuple := types.NewTupleType([]eval.Type{types.DefaultStringType(), types.DefaultIntegerType()}, nil)
	fmt.Println(tuple)
	// Output: Tuple[String, Integer]
}

func ExampleWrapHash() {
	a := eval.Wrap(nil, map[string]interface{}{
		`foo`: 23,
		`fee`: `hello`,
		`fum`: map[string]interface{}{
			`x`: `1`,
			`y`: []int{1, 2, 3},
			`z`: regexp.MustCompile(`^[a-z]+$`)}})

	e := types.WrapHash([]*types.HashEntry{
		types.WrapHashEntry2(`foo`, types.WrapInteger(23)),
		types.WrapHashEntry2(`fee`, types.WrapString(`hello`)),
		types.WrapHashEntry2(`fum`, types.WrapHash([]*types.HashEntry{
			types.WrapHashEntry2(`x`, types.WrapString(`1`)),
			types.WrapHashEntry2(`y`, types.WrapValues([]eval.Value{
				types.WrapInteger(1), types.WrapInteger(2), types.WrapInteger(3)})),
			types.WrapHashEntry2(`z`, types.WrapRegexp(`^[a-z]+$`))}))})

	fmt.Println(eval.Equals(e, a))
	// Output: true
}

func TestIsAssignable(t *testing.T) {
	t1 := &types.AnyType{}
	t2 := &types.UnitType{}
	if !eval.IsAssignable(t1, t2) {
		t.Error(`Unit not assignable to Any`)
	}
}
