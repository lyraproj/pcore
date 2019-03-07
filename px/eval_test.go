package px_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/lyraproj/pcore/pcore"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/semver/semver"
)

func ExampleContext_ParseType2() {
	pcore.Do(func(c px.Context) {
		t := c.ParseType2(`Object[
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

func ExampleContext_ParseType2_enum() {
	pcore.Do(func(ctx px.Context) {
		pcoreType := ctx.ParseType2("Enum[foo,fee,fum]")
		fmt.Printf("%s is an instance of %s\n", pcoreType, pcoreType.PType())
	})
	// Output:
	// Enum['foo', 'fee', 'fum'] is an instance of Type[Enum['foo', 'fee', 'fum']]
}

func ExampleIsInstance() {
	pcore.Do(func(ctx px.Context) {
		pcoreType := ctx.ParseType2("Enum[foo,fee,fum]")
		fmt.Println(px.IsInstance(pcoreType, px.Wrap(ctx, "foo")))
		fmt.Println(px.IsInstance(pcoreType, px.Wrap(ctx, "bar")))
	})
	// Output:
	// true
	// false
}

func ExampleContext_ParseType2_error() {
	err := pcore.Try(func(ctx px.Context) error {
		ctx.ParseType2("Enum[foo") // Missing end bracket
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: expected one of ',' or ']', got 'EOF' (line: 1, column: 9)
}

func ExampleObjectType_fromReflectedValue() {
	type TestStruct struct {
		Message   string
		Kind      string
		IssueCode string `puppet:"name => issue_code"`
	}

	c := pcore.RootContext()
	ts := &TestStruct{`the message`, `THE_KIND`, `THE_CODE`}
	et, _ := px.Load(c, px.NewTypedName(px.NsType, `Error`))
	ev := et.(px.ObjectType).FromReflectedValue(c, reflect.ValueOf(ts).Elem())
	fmt.Println(ev)
	// Output: Error('message' => 'the message', 'kind' => 'THE_KIND', 'issue_code' => 'THE_CODE')
}

func ExampleImplementationRegistry() {
	type TestAddress struct {
		Street string
		Zip    string
	}
	type TestPerson struct {
		Name    string
		Age     int
		Address *TestAddress
		Active  bool
	}

	address, err := types.Parse(`
    attributes => {
      street => String,
      zip => String,
    }`)
	if err != nil {
		panic(err)
	}
	person, err := types.Parse(`
		attributes => {
      name => String,
      age => Integer,
      address => My::Address,
      active => Boolean,
		}`)
	if err != nil {
		panic(err)
	}

	c := pcore.RootContext()
	px.AddTypes(c, types.NamedType(``, `My::Address`, address), types.NamedType(``, `My::Person`, person))

	ir := c.ImplementationRegistry()
	ir.RegisterType(c.ParseType2(`My::Address`), reflect.TypeOf(TestAddress{}))
	ir.RegisterType(c.ParseType2(`My::Person`), reflect.TypeOf(TestPerson{}))

	ts := &TestPerson{`Bob Tester`, 34, &TestAddress{`Example Road 23`, `12345`}, true}
	ev := px.Wrap(c, ts)
	fmt.Println(ev)
	// Output: My::Person('name' => 'Bob Tester', 'age' => 34, 'address' => My::Address('street' => 'Example Road 23', 'zip' => '12345'), 'active' => true)
}

func ExampleImplementationRegistry_tags() {
	type TestAddress struct {
		Street string
		Zip    string `puppet:"name=>zip_code"`
	}
	type TestPerson struct {
		Name    string
		Age     int
		Address *TestAddress
		Active  bool `puppet:"name=>enabled"`
	}

	address, _ := types.Parse(`
    attributes => {
      street => String,
      zip_code => Optional[String],
    }`)

	person, _ := types.Parse(`
		attributes => {
      name => String,
      age => Integer,
      address => My::Address,
      enabled => Boolean,
		}`)

	c := pcore.RootContext()
	px.AddTypes(c, types.NamedType(``, `My::Address`, address), types.NamedType(``, `My::Person`, person))

	ir := c.ImplementationRegistry()
	ir.RegisterType(c.ParseType2(`My::Address`), reflect.TypeOf(TestAddress{}))
	ir.RegisterType(c.ParseType2(`My::Person`), reflect.TypeOf(TestPerson{}))

	ts := &TestPerson{`Bob Tester`, 34, &TestAddress{`Example Road 23`, `12345`}, true}
	ev := px.Wrap(c, ts)
	fmt.Println(ev)
	// Output: My::Person('name' => 'Bob Tester', 'age' => 34, 'address' => My::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'enabled' => true)
}

func TestReflectorAndImplRepo(t *testing.T) {
	type ObscurelyNamedAddress struct {
		Street string
		Zip    string `puppet:"name=>zip_code"`
	}
	type Person struct {
		Name    string
		Address ObscurelyNamedAddress
	}

	pcore.Do(func(c px.Context) {
		typeSet := c.Reflector().TypeSetFromReflect(`My`, semver.MustParseVersion(`1.0.0`), map[string]string{`ObscurelyNamedAddress`: `Address`},
			reflect.TypeOf(&ObscurelyNamedAddress{}), reflect.TypeOf(&Person{}))
		px.AddTypes(c, typeSet)
		tss := typeSet.String()
		exp := `TypeSet[{pcore_uri => 'http://puppet.com/2016.1/pcore', pcore_version => '1.0.0', name_authority => 'http://puppet.com/2016.1/runtime', name => 'My', version => '1.0.0', types => {Address => {attributes => {'street' => String, 'zip_code' => String}}, Person => {attributes => {'name' => String, 'address' => Address}}}}]`
		if tss != exp {
			t.Errorf("Expected %s, got %s\n", exp, tss)
		}
	})
}
