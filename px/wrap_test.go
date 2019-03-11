package px_test

import (
	"fmt"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
)

func ExampleWrap() {
	// Wrap native Go types
	str := px.Wrap(nil, "hello")
	idx := px.Wrap(nil, 23)
	bl := px.Wrap(nil, true)
	und := px.Undef

	fmt.Printf("'%s' is a %s\n", str, str.PType())
	fmt.Printf("'%s' is a %s\n", idx, idx.PType())
	fmt.Printf("'%s' is a %s\n", bl, bl.PType())
	fmt.Printf("'%s' is a %s\n", und, und.PType())

	// Output:
	// 'hello' is a String
	// '23' is a Integer[23, 23]
	// 'true' is a Boolean[true]
	// 'undef' is a Undef
}

func ExampleWrap_slice() {
	// Wrap native Go slice
	arr := px.Wrap(nil, []interface{}{1, "2", true, nil, "hello"})
	fmt.Printf("%s is an %s\n", arr, arr.PType())

	// Output:
	// [1, '2', true, undef, 'hello'] is an Array[Data, 5, 5]
}

func ExampleWrap_hash() {
	// Wrap native Go hash
	hsh := px.Wrap(nil, map[string]interface{}{
		"first":  1,
		"second": 20,
		"third":  "three",
		"nested": []string{"hello", "world"},
	})
	nst, _ := hsh.(px.OrderedMap).Get4("nested")
	fmt.Printf("'%s' is a %s\n", hsh, hsh.PType())
	fmt.Printf("hsh['nested'] == %s, an instance of %s\n", nst, nst.PType())

	// Output:
	// '{'first' => 1, 'nested' => ['hello', 'world'], 'second' => 20, 'third' => 'three'}' is a Hash[Enum['first', 'nested', 'second', 'third'], Data, 4, 4]
	// hsh['nested'] == ['hello', 'world'], an instance of Array[Enum['hello', 'world'], 2, 2]
}

func ExampleIsInstance() {
	pcore.Do(func(ctx px.Context) {
		pcoreType := ctx.ParseType("Enum[foo,fee,fum]")
		fmt.Println(px.IsInstance(pcoreType, px.Wrap(ctx, "foo")))
		fmt.Println(px.IsInstance(pcoreType, px.Wrap(ctx, "bar")))
	})
	// Output:
	// true
	// false
}
