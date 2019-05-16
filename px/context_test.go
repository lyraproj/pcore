package px_test

import (
	"fmt"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
)

func ExampleContext_ParseType() {
	pcore.Do(func(c px.Context) {
		t := c.ParseType(`Object[
      name => 'Address',
      attributes => {
        'annotations' => {
          'type' => Optional[Hash[String, String]],
          'value' => undef
        },
        'lineOne' => {
          'type' => String,
          'value' => ''
        }
      }
    ]`)
		px.AddTypes(c, t)
		v := px.New(c, t, px.Wrap(c, map[string]string{`lineOne`: `30 East 60th Street`}))
		fmt.Println(v.String())
	})

	// Output: Address('lineOne' => '30 East 60th Street')
}

func ExampleContext_ParseType_enum() {
	pcore.Do(func(ctx px.Context) {
		pcoreType := ctx.ParseType("Enum[foo,fee,fum]")
		fmt.Printf("%s is an instance of %s\n", pcoreType, pcoreType.PType())
	})
	// Output:
	// Enum['foo', 'fee', 'fum'] is an instance of Type[Enum['foo', 'fee', 'fum']]
}

func ExampleContext_ParseType_error() {
	err := pcore.Try(func(ctx px.Context) error {
		ctx.ParseType("Enum[foo") // Missing end bracket
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: expected one of ',' or ']', got 'EOF' (file: <pcore type expression>, line: 1, column: 9)
}
