package types_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func ExampleParse_qName() {
	t := types.Parse(`Foo::Bar`)
	t.ToString(os.Stdout, px.PrettyExpanded, nil)
	fmt.Println()
	// Output: DeferredType(Foo::Bar)
}

func ExampleParse_int() {
	t := types.Parse(`23`)
	t.ToString(os.Stdout, px.PrettyExpanded, nil)
	fmt.Println()
	// Output: 23
}

func ExampleParse_entry() {
	const src = `# This is scanned code.
    constants => {
      first => 0,
      second => 0x32,
      third => 2e4,
      fourth => 2.3e-2,
      fifth => 'hello',
      sixth => "world",
      type => Foo::Bar[1,'23',Baz[1,2]],
      value => "String\nWith \\Escape",
      array => [a, b, c],
      call => Boo::Bar("with", "args")
    }
  `
	v := types.Parse(src)
	v.ToString(os.Stdout, px.PrettyExpanded, nil)
	fmt.Println()
	// Output:
	// {
	//   'constants' => {
	//     'first' => 0,
	//     'second' => 50,
	//     'third' => 20000.0,
	//     'fourth' => 0.02300,
	//     'fifth' => 'hello',
	//     'sixth' => 'world',
	//     'type' => DeferredType(Foo::Bar, [1, '23', DeferredType(Baz, [1, 2])]),
	//     'value' => "String\nWith \\Escape",
	//     'array' => ['a', 'b', 'c'],
	//     'call' => Deferred(
	//       'name' => 'new',
	//       'arguments' => ['Boo::Bar', 'with', 'args']
	//     )
	//   }
	// }
	//
}

func ExampleParse_hash() {
	v := types.Parse(`{value=>-1}`)
	fmt.Println(v)
	// Output: {'value' => -1}
}

func TestParse_emptyTypeArgs(t *testing.T) {
	requireError(t, `expected a literal, got ']' (file: <pcore type expression>, line: 1, column: 7)`, func() { types.Parse(`String[]`) })
}

func requireError(t *testing.T, msg string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				require.Equal(t, msg, err.Error())
			} else {
				panic(r)
			}
		}
	}()
	f()
	require.Fail(t, `expected panic didn't happen`)
}
