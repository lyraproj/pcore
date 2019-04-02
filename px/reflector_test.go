package px_test

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/semver/semver"
)

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

	pcore.Do(func(c px.Context) {
		px.AddTypes(c, types.NamedType(``, `My::Address`, address), types.NamedType(``, `My::Person`, person))

		ir := c.ImplementationRegistry()
		ir.RegisterType(c.ParseType(`My::Address`), reflect.TypeOf(TestAddress{}))
		ir.RegisterType(c.ParseType(`My::Person`), reflect.TypeOf(TestPerson{}))

		ts := &TestPerson{`Bob Tester`, 34, &TestAddress{`Example Road 23`, `12345`}, true}
		ev := px.Wrap(c, ts)
		fmt.Println(ev)
	})
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

	pcore.Do(func(c px.Context) {
		px.AddTypes(c, types.NamedType(``, `My::Address`, address), types.NamedType(``, `My::Person`, person))

		ir := c.ImplementationRegistry()
		ir.RegisterType(c.ParseType(`My::Address`), reflect.TypeOf(TestAddress{}))
		ir.RegisterType(c.ParseType(`My::Person`), reflect.TypeOf(TestPerson{}))

		ts := &TestPerson{`Bob Tester`, 34, &TestAddress{`Example Road 23`, `12345`}, true}
		ev := px.Wrap(c, ts)
		fmt.Println(ev)
	})
	// Output: My::Person('name' => 'Bob Tester', 'age' => 34, 'address' => My::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'enabled' => true)
}

func TestReflectorAndImplRepo(t *testing.T) {
	type ObscurelyNamedAddress struct {
		Street string
		Zip    string `puppet:"name=>zip_code"`
	}
	type Person struct {
		Name    string
		Gender  string `puppet:"type=>Enum[male,female,other]"`
		Address ObscurelyNamedAddress
	}

	pcore.Do(func(c px.Context) {
		typeSet := c.Reflector().TypeSetFromReflect(`My`, semver.MustParseVersion(`1.0.0`), map[string]string{`ObscurelyNamedAddress`: `Address`},
			reflect.TypeOf(&ObscurelyNamedAddress{}), reflect.TypeOf(&Person{}))
		px.AddTypes(c, typeSet)
		tss := typeSet.String()
		exp := `TypeSet[{pcore_uri => 'http://puppet.com/2016.1/pcore', pcore_version => '1.0.0', name_authority => 'http://puppet.com/2016.1/runtime', name => 'My', version => '1.0.0', types => {Address => {attributes => {'street' => String, 'zip_code' => String}}, Person => {attributes => {'name' => String, 'gender' => Enum['male', 'female', 'other'], 'address' => Address}}}}]`
		if tss != exp {
			t.Errorf("Expected %s, got %s\n", exp, tss)
		}
	})
}

func ExampleReflector_reflectArray() {
	pcore.Do(func(c px.Context) {
		av := px.Wrap(nil, []interface{}{`hello`, 3}).(*types.Array)
		ar := c.Reflector().Reflect(av)
		fmt.Printf("%s %v\n", ar.Type(), ar)

		av = px.Wrap(nil, []interface{}{`hello`, `world`}).(*types.Array)
		ar = c.Reflector().Reflect(av)
		fmt.Printf("%s %v\n", ar.Type(), ar)
	})
	// Output:
	// []interface {} [hello 3]
	// []string [hello world]
}

func ExampleReflector_reflectToArray() {
	type TestStruct struct {
		Strings    []string
		Interfaces []interface{}
		Values     []px.Value
	}
	pcore.Do(func(c px.Context) {
		ts := &TestStruct{}
		rv := reflect.ValueOf(ts).Elem()

		av := px.Wrap(nil, []string{`hello`, `world`})

		rf := c.Reflector()
		rf.ReflectTo(av, rv.Field(0))
		rf.ReflectTo(av, rv.Field(1))
		rf.ReflectTo(av, rv.Field(2))
		fmt.Println(ts)

		rf.ReflectTo(px.Undef, rv.Field(0))
		rf.ReflectTo(px.Undef, rv.Field(1))
		rf.ReflectTo(px.Undef, rv.Field(2))
		fmt.Println(ts)
	})
	// Output:
	// &{[hello world] [hello world] [hello world]}
	// &{[] [] []}
}

func ExampleReflector_reflectToHash() {
	var strings map[string]string
	var interfaces map[string]interface{}
	var values map[string]px.Value

	pcore.Do(func(c px.Context) {
		hv := px.Wrap(nil, map[string]string{`x`: `hello`, `y`: `world`})

		rf := c.Reflector()
		rf.ReflectTo(hv, reflect.ValueOf(&strings).Elem())
		rf.ReflectTo(hv, reflect.ValueOf(&interfaces).Elem())
		rf.ReflectTo(hv, reflect.ValueOf(&values).Elem())
		fmt.Printf("%s %s\n", strings[`x`], strings[`y`])
		fmt.Printf("%s %s\n", interfaces[`x`], interfaces[`y`])
		fmt.Printf("%s %s\n", values[`x`], values[`y`])

		rf.ReflectTo(px.Undef, reflect.ValueOf(&strings).Elem())
		rf.ReflectTo(px.Undef, reflect.ValueOf(&interfaces).Elem())
		rf.ReflectTo(px.Undef, reflect.ValueOf(&values).Elem())
		fmt.Println(strings)
		fmt.Println(interfaces)
		fmt.Println(values)
	})
	// Output:
	// hello world
	// hello world
	// hello world
	// map[]
	// map[]
	// map[]
}

func ExampleReflector_reflectToBytes() {
	var buf []byte

	pcore.Do(func(c px.Context) {
		rf := c.Reflector()
		bv := px.Wrap(nil, []byte{1, 2, 3})
		rf.ReflectTo(bv, reflect.ValueOf(&buf).Elem())
		fmt.Println(buf)

		rf.ReflectTo(px.Undef, reflect.ValueOf(&buf).Elem())
		fmt.Println(buf)
	})
	// Output:
	// [1 2 3]
	// []
}

func ExampleReflector_reflectToFloat() {
	type TestStruct struct {
		SmallFloat float32
		BigFloat   float64
		APValue    px.Value
		IValue     interface{}
	}

	pcore.Do(func(c px.Context) {
		rf := c.Reflector()
		ts := &TestStruct{}
		rv := reflect.ValueOf(ts).Elem()
		n := rv.NumField()
		for i := 0; i < n; i++ {
			fv := px.Wrap(nil, float64(10+i+1)/10.0)
			rf.ReflectTo(fv, rv.Field(i))
		}
		fmt.Println(ts)
	})
	// Output: &{1.1 1.2 1.3 1.4}
}

func ExampleReflector_reflectToInt() {
	type TestStruct struct {
		AnInt    int
		AnInt8   int8
		AnInt16  int16
		AnInt32  int32
		AnInt64  int64
		AnUInt   uint
		AnUInt8  uint8
		AnUInt16 uint16
		AnUInt32 uint32
		AnUInt64 uint64
		APValue  px.Value
		IValue   interface{}
	}

	pcore.Do(func(c px.Context) {
		rf := c.Reflector()
		ts := &TestStruct{}
		rv := reflect.ValueOf(ts).Elem()
		n := rv.NumField()
		for i := 0; i < n; i++ {
			rf.ReflectTo(px.Wrap(nil, 10+i), rv.Field(i))
		}
		fmt.Println(ts)
	})
	// Output: &{10 11 12 13 14 15 16 17 18 19 20 21}
}

func ExampleReflector_reflectToRegexp() {
	pcore.Do(func(c px.Context) {
		var expr regexp.Regexp

		rx := px.Wrap(c, regexp.MustCompile("[a-z]*"))
		c.Reflector().ReflectTo(rx, reflect.ValueOf(&expr).Elem())

		fmt.Println(expr.String())
	})
	// Output: [a-z]*
}

func ExampleReflector_reflectToTimespan() {
	pcore.Do(func(c px.Context) {
		var span time.Duration

		rx := px.Wrap(c, time.Duration(1234))
		c.Reflector().ReflectTo(rx, reflect.ValueOf(&span).Elem())

		fmt.Println(span)
	})
	// Output: 1.234µs
}

func ExampleReflector_reflectToTimestamp() {
	pcore.Do(func(c px.Context) {
		var tx time.Time

		tm, _ := time.Parse(time.RFC3339, "2018-05-11T06:31:22+01:00")
		tv := px.Wrap(c, tm)
		c.Reflector().ReflectTo(tv, reflect.ValueOf(&tx).Elem())

		fmt.Println(tx.Format(time.RFC3339))
	})
	// Output: 2018-05-11T06:31:22+01:00
}

func ExampleReflector_reflectToVersion() {
	pcore.Do(func(c px.Context) {
		var version semver.Version

		ver, _ := semver.ParseVersion(`1.2.3`)
		vv := px.Wrap(c, ver)
		c.Reflector().ReflectTo(vv, reflect.ValueOf(&version).Elem())

		fmt.Println(version)
	})
	// Output: 1.2.3
}

func ExampleReflector_typeFromReflect() {
	type TestAddress struct {
		Street string
		Zip    string
	}
	type TestPerson struct {
		Name    string
		Address *TestAddress
	}
	type TestExtendedPerson struct {
		TestPerson
		Age    *int `other:"stuff"`
		Active bool `puppet:"name=>enabled"`
	}
	pcore.Do(func(c px.Context) {
		rtAddress := reflect.TypeOf(&TestAddress{})
		rtPerson := reflect.TypeOf(&TestPerson{})
		rtExtPerson := reflect.TypeOf(&TestExtendedPerson{})

		rf := c.Reflector()
		tAddress := rf.TypeFromTagged(`My::Address`, nil, px.NewTaggedType(rtAddress, map[string]string{`Zip`: `name=>zip_code`}), nil)
		tPerson := rf.TypeFromReflect(`My::Person`, nil, rtPerson)
		tExtPerson := rf.TypeFromReflect(`My::ExtendedPerson`, tPerson, rtExtPerson)
		px.AddTypes(c, tAddress, tPerson, tExtPerson)

		tAddress.ToString(os.Stdout, types.Expanded, nil)
		fmt.Println()
		tPerson.ToString(os.Stdout, types.Expanded, nil)
		fmt.Println()
		tExtPerson.ToString(os.Stdout, types.Expanded, nil)
		fmt.Println()

		age := 34
		ts := &TestExtendedPerson{TestPerson{`Bob Tester`, &TestAddress{`Example Road 23`, `12345`}}, &age, true}
		ev := px.Wrap(c, ts)
		fmt.Println(ev)
	})
	// Output:
	// Object[{name => 'My::Address', attributes => {'street' => String, 'zip_code' => String}}]
	// Object[{name => 'My::Person', attributes => {'name' => String, 'address' => Optional[My::Address]}}]
	// Object[{name => 'My::ExtendedPerson', parent => My::Person, attributes => {'age' => {'annotations' => {TagsAnnotation => {'other' => 'stuff'}}, 'type' => Optional[Integer]}, 'enabled' => Boolean}}]
	// My::ExtendedPerson('name' => 'Bob Tester', 'enabled' => true, 'address' => My::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'age' => 34)
}

type A interface {
	Int() int
	X() string
	Y(A) A
}

type B int

func NewA(x int) A {
	return B(x)
}

func (b B) X() string {
	return strconv.Itoa(int(b) + 5)
}

func (b B) Int() int {
	return int(b)
}

func (b B) Y(a A) A {
	return NewA(int(b) + a.Int())
}

func ExampleReflector_typeFromReflectInterface() {
	pcore.Do(func(c px.Context) {
		// Create ObjectType from reflected type
		xa := c.Reflector().TypeFromReflect(`X::A`, nil, reflect.TypeOf((*A)(nil)).Elem())
		xb := c.Reflector().TypeFromReflect(`X::B`, nil, reflect.TypeOf(B(0)))

		// Ensure that the type is resolved
		px.AddTypes(c, xa, xb)

		// Print the created Interface Type in human readable form
		xa.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()

		// Print the created Implementation Type in human readable form
		xb.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()

		// Invoke method 'x' on the interface on a receiver
		m, _ := xb.Member(`x`)
		gm := m.(px.CallableGoMember)

		fmt.Println(gm.CallGo(c, NewA(32))[0])
		fmt.Println(gm.CallGo(c, B(25))[0])

		// Invoke method 'x' on the interface on a receiver
		m, _ = xb.Member(`y`)

		// Call Go function using CallableGoMember
		gv := m.(px.CallableGoMember).CallGo(c, B(25), NewA(15))[0]
		fmt.Printf("%T %v\n", gv, gv)

		// Call Go function using CallableMember and pcore.Value
		pv := m.Call(c, px.New(c, xb, px.Wrap(c, 25)), nil, []px.Value{px.New(c, xb, px.Wrap(c, 15))})
		fmt.Println(pv)
	})

	// Output:
	// Object[{
	//   name => 'X::A',
	//   functions => {
	//     'int' => Callable[
	//       [0, 0],
	//       Integer],
	//     'x' => Callable[
	//       [0, 0],
	//       String],
	//     'y' => Callable[
	//       [X::A],
	//       X::A]
	//   }
	// }]
	// Object[{
	//   name => 'X::B',
	//   attributes => {
	//     'value' => Integer
	//   },
	//   functions => {
	//     'int' => Callable[
	//       [0, 0],
	//       Integer],
	//     'x' => Callable[
	//       [0, 0],
	//       String],
	//     'y' => Callable[
	//       [X::A],
	//       X::A]
	//   }
	// }]
	// 37
	// 30
	// px_test.B 40
	// X::B('value' => 40)
}

type Address struct {
	Street string
	Zip    string `puppet:"name=>zip_code"`
}
type Person struct {
	Name    string
	Address *Address
}
type ExtendedPerson struct {
	Person
	Birth          *time.Time
	TimeSinceVisit *time.Duration
	Active         bool `puppet:"name=>enabled"`
}

func (p *Person) Visit(v *Address) string {
	return "visited " + v.Street
}

func ExampleReflector_typeSetFromReflect() {
	pcore.Do(func(c px.Context) {
		// Create a TypeSet from a list of Go structs
		typeSet := c.Reflector().TypeSetFromReflect(`My::Own`, semver.MustParseVersion(`1.0.0`), nil,
			reflect.TypeOf(&Address{}), reflect.TypeOf(&Person{}), reflect.TypeOf(&ExtendedPerson{}))

		// Make the types known to the current loader
		px.AddTypes(c, typeSet)

		// Print the TypeSet in human readable form
		typeSet.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()

		// Create an instance of something included in the TypeSet
		ad := &Address{`Example Road 23`, `12345`}
		birth, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		tsv, _ := time.ParseDuration("1h20m")
		ep := &ExtendedPerson{Person{`Bob Tester`, ad}, &birth, &tsv, true}

		// Wrap the instance as a Value and print it
		v := px.Wrap(c, ep)
		fmt.Println(v)

		m, _ := v.PType().(px.TypeWithCallableMembers).Member(`visit`)
		fmt.Println(m.(px.CallableGoMember).CallGo(c, ep, ad)[0])
		fmt.Println(m.Call(c, v, nil, []px.Value{px.Wrap(c, ad)}))
	})

	// Output:
	// TypeSet[{
	//   pcore_uri => 'http://puppet.com/2016.1/pcore',
	//   pcore_version => '1.0.0',
	//   name_authority => 'http://puppet.com/2016.1/runtime',
	//   name => 'My::Own',
	//   version => '1.0.0',
	//   types => {
	//     Address => {
	//       attributes => {
	//         'street' => String,
	//         'zip_code' => String
	//       }
	//     },
	//     Person => {
	//       attributes => {
	//         'name' => String,
	//         'address' => Optional[Address]
	//       },
	//       functions => {
	//         'visit' => Callable[
	//           [Optional[Address]],
	//           String]
	//       }
	//     },
	//     ExtendedPerson => Person{
	//       attributes => {
	//         'birth' => Optional[Timestamp],
	//         'timeSinceVisit' => Optional[Timespan],
	//         'enabled' => Boolean
	//       }
	//     }
	//   }
	// }]
	// My::Own::ExtendedPerson('name' => 'Bob Tester', 'enabled' => true, 'address' => My::Own::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'birth' => 2006-01-02T15:04:05.000000000 UTC, 'timeSinceVisit' => 0-01:20:00.0)
	// visited Example Road 23
	// visited Example Road 23
}

type twoValueReturn struct {
}

func (t *twoValueReturn) ReturnTwo() (string, int) {
	return "number", 42
}

func (t *twoValueReturn) ReturnTwoAndErrorOk() (string, int, error) {
	return "number", 42, nil
}

func (t *twoValueReturn) ReturnTwoAndErrorFail() (string, int, error) {
	return ``, 0, fmt.Errorf(`bad things happened`)
}

func ExampleReflector_twoValueReturn() {
	pcore.Do(func(c px.Context) {
		gv := &twoValueReturn{}
		api := c.Reflector().TypeFromReflect(`A::B`, nil, reflect.TypeOf(gv))
		px.AddTypes(c, api)
		v := px.Wrap(c, gv)
		r, _ := api.Member(`returnTwo`)
		fmt.Println(px.ToPrettyString(r.Call(c, v, nil, px.EmptyValues)))
	})
	// Output:
	// ['number', 42]
}

func ExampleReflector_TypeFromReflect_twoValueReturnErrorOk() {
	pcore.Do(func(c px.Context) {
		gv := &twoValueReturn{}
		api := c.Reflector().TypeFromReflect(`A::B`, nil, reflect.TypeOf(gv))
		px.AddTypes(c, api)
		v := px.Wrap(c, gv)
		if r, ok := api.Member(`returnTwoAndErrorOk`); ok {
			fmt.Println(px.ToPrettyString(r.Call(c, v, nil, px.EmptyValues)))
		}
	})
	// Output:
	// ['number', 42]
}

func ExampleReflector_TypeFromReflect_twoValueReturnErrorFail() {
	err := pcore.Try(func(c px.Context) error {
		gv := &twoValueReturn{}
		api := c.Reflector().TypeFromReflect(`A::B`, nil, reflect.TypeOf(gv))
		px.AddTypes(c, api)
		v := px.Wrap(c, gv)
		if r, ok := api.Member(`returnTwoAndErrorFail`); ok {
			r.Call(c, v, nil, px.EmptyValues)
		}
		return nil
	})
	if err != nil {
		fmt.Println(err.Error()[:70])
	}
	// Output:
	// Go function ReturnTwoAndErrorFail returned error 'bad things happened'
}

type valueStruct struct {
	X px.OrderedMap
	Y *types.Array
	P px.PuppetObject
	O px.Object
}

func (v *valueStruct) Get(key px.Integer, dflt px.Value) px.StringValue {
	return v.X.Get2(key, px.Undef).(px.StringValue)
}

func ExampleReflector_reflectPType() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X::M`, nil, reflect.TypeOf(&valueStruct{}))
		px.AddTypes(c, xm)
		xm.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})
	// Output:
	// Object[{
	//   name => 'X::M',
	//   attributes => {
	//     'x' => Hash,
	//     'y' => Array,
	//     'p' => Object,
	//     'o' => Object
	//   },
	//   functions => {
	//     'get' => Callable[
	//       [Integer, Any],
	//       String]
	//   }
	// }]
}

type optionalString struct {
	A string
	B *string
}

func ExampleReflector_TypeFromReflect_optionalString() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		px.AddTypes(c, xm)
		xm.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => String,
	//     'b' => Optional[String]
	//   }
	// }]
}

func ExampleReflector_TypeFromReflect_optionalStringReflect() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		px.AddTypes(c, xm)
		w := `World`
		fmt.Println(px.ToPrettyString(px.Wrap(c, &optionalString{`Hello`, &w})))
	})

	// Output:
	// X(
	//   'a' => 'Hello',
	//   'b' => 'World'
	// )
}

func ExampleReflector_TypeFromReflect_optionalStringReflectZero() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		px.AddTypes(c, xm)
		var w string
		fmt.Println(px.ToPrettyString(px.Wrap(c, &optionalString{`Hello`, &w})))
	})

	// Output:
	// X(
	//   'a' => 'Hello',
	//   'b' => ''
	// )
}

func ExampleReflector_TypeFromReflect_optionalStringCreate() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		px.AddTypes(c, xm)
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapString(`Hello`), types.WrapString(`World`))))
	})

	// Output:
	// X(
	//   'a' => 'Hello',
	//   'b' => 'World'
	// )
}

func ExampleReflector_TypeFromReflect_optionalStringDefault() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		px.AddTypes(c, xm)
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapString(`Hello`))))
	})

	// Output:
	// X(
	//   'a' => 'Hello'
	// )
}

func ExampleReflector_TypeFromReflect_optionalStringDefaultZero() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		px.AddTypes(c, xm)
		var w string
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapString(`Hello`), types.WrapString(w))))
	})

	// Output:
	// X(
	//   'a' => 'Hello',
	//   'b' => ''
	// )
}

type optionalBoolean struct {
	A bool
	B *bool
}

func ExampleReflector_TypeFromReflect_optionalBoolean() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		px.AddTypes(c, xm)
		xm.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => Boolean,
	//     'b' => Optional[Boolean]
	//   }
	// }]
}

func ExampleReflector_TypeFromReflect_optionalBooleanReflect() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		px.AddTypes(c, xm)
		w := true
		fmt.Println(px.ToPrettyString(px.Wrap(c, &optionalBoolean{true, &w})))
	})

	// Output:
	// X(
	//   'a' => true,
	//   'b' => true
	// )
}

func ExampleReflector_TypeFromReflect_optionalBooleanReflectZero() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		px.AddTypes(c, xm)
		var w bool
		fmt.Println(px.ToPrettyString(px.Wrap(c, &optionalBoolean{true, &w})))
	})

	// Output:
	// X(
	//   'a' => true,
	//   'b' => false
	// )
}

func ExampleReflector_TypeFromReflect_optionalBooleanCreate() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		px.AddTypes(c, xm)
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapBoolean(true), types.WrapBoolean(true))))
	})

	// Output:
	// X(
	//   'a' => true,
	//   'b' => true
	// )
}

func ExampleReflector_TypeFromReflect_optionalBooleanDefault() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		px.AddTypes(c, xm)
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapBoolean(true))))
	})

	// Output:
	// X(
	//   'a' => true
	// )
}

func ExampleReflector_TypeFromReflect_optionalBooleanZero() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		px.AddTypes(c, xm)
		var w bool
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapBoolean(true), types.WrapBoolean(w))))
	})

	// Output:
	// X(
	//   'a' => true,
	//   'b' => false
	// )
}

type optionalInt struct {
	A int64
	B *int64
	C *byte
	D *int16
	E *uint32
}

func ExampleReflector_TypeFromReflect_optionalInt() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		px.AddTypes(c, xm)
		xm.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => Integer,
	//     'b' => Optional[Integer],
	//     'c' => Optional[Integer[0, 255]],
	//     'd' => Optional[Integer[-32768, 32767]],
	//     'e' => Optional[Integer[0, 4294967295]]
	//   }
	// }]
}

func ExampleReflector_TypeFromReflect_optionalIntReflect() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		px.AddTypes(c, xm)
		w1 := int64(2)
		w2 := byte(3)
		w3 := int16(4)
		w4 := uint32(5)
		fmt.Println(px.ToPrettyString(px.Wrap(c, &optionalInt{1, &w1, &w2, &w3, &w4})))
	})

	// Output:
	// X(
	//   'a' => 1,
	//   'b' => 2,
	//   'c' => 3,
	//   'd' => 4,
	//   'e' => 5
	// )
}

func ExampleReflector_TypeFromReflect_optionalIntReflectZero() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		px.AddTypes(c, xm)
		var w1 int64
		var w2 byte
		var w3 int16
		var w4 uint32
		fmt.Println(px.ToPrettyString(px.Wrap(c, &optionalInt{1, &w1, &w2, &w3, &w4})))
	})

	// Output:
	// X(
	//   'a' => 1,
	//   'b' => 0,
	//   'c' => 0,
	//   'd' => 0,
	//   'e' => 0
	// )
}

func ExampleReflector_TypeFromReflect_optionalIntCreate() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		px.AddTypes(c, xm)
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapInteger(1), types.WrapInteger(2), types.WrapInteger(3), types.WrapInteger(4), types.WrapInteger(5))))
	})

	// Output:
	// X(
	//   'a' => 1,
	//   'b' => 2,
	//   'c' => 3,
	//   'd' => 4,
	//   'e' => 5
	// )
}

func ExampleReflector_TypeFromReflect_optionalIntDefault() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		px.AddTypes(c, xm)
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapInteger(1))))
	})

	// Output:
	// X(
	//   'a' => 1
	// )
}

func ExampleReflector_TypeFromReflect_optionalIntDefaultZero() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		px.AddTypes(c, xm)
		var w1 int64
		var w2 byte
		var w3 int16
		var w4 uint32
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapInteger(1), types.WrapInteger(w1), types.WrapInteger(int64(w2)), types.WrapInteger(int64(w3)), types.WrapInteger(int64(w4)))))
	})

	// Output:
	// X(
	//   'a' => 1,
	//   'b' => 0,
	//   'c' => 0,
	//   'd' => 0,
	//   'e' => 0
	// )
}

type optionalFloat struct {
	A float64
	B *float64
	C *float32
}

func ExampleReflector_TypeFromReflect_optionalFloat() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		px.AddTypes(c, xm)
		xm.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => Float,
	//     'b' => Optional[Float],
	//     'c' => Optional[Float[-3.4028234663852886e+38, 3.4028234663852886e+38]]
	//   }
	// }]
}

func ExampleReflector_TypeFromReflect_optionalFloatReflect() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		px.AddTypes(c, xm)
		w1 := float64(2)
		w2 := float32(3)
		fmt.Println(px.ToPrettyString(px.Wrap(c, &optionalFloat{1, &w1, &w2})))
	})

	// Output:
	// X(
	//   'a' => 1.00000,
	//   'b' => 2.00000,
	//   'c' => 3.00000
	// )
}

func ExampleReflector_TypeFromReflect_optionalFloatReflectZero() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		px.AddTypes(c, xm)
		var w1 float64
		var w2 float32
		fmt.Println(px.ToPrettyString(px.Wrap(c, &optionalFloat{1, &w1, &w2})))
	})

	// Output:
	// X(
	//   'a' => 1.00000,
	//   'b' => 0.00000,
	//   'c' => 0.00000
	// )
}

func ExampleReflector_TypeFromReflect_optionalFloatCreate() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		px.AddTypes(c, xm)
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapFloat(1), types.WrapFloat(2), types.WrapFloat(3))))
	})

	// Output:
	// X(
	//   'a' => 1.00000,
	//   'b' => 2.00000,
	//   'c' => 3.00000
	// )
}

func ExampleReflector_TypeFromReflect_optionalFloatDefault() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		px.AddTypes(c, xm)
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapFloat(1))))
	})

	// Output:
	// X(
	//   'a' => 1.00000
	// )
}

func ExampleReflector_TypeFromReflect_optionalFloatZero() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		px.AddTypes(c, xm)
		var w1 float64
		var w2 float32
		fmt.Println(px.ToPrettyString(px.New(c, xm, types.WrapFloat(1), types.WrapFloat(w1), types.WrapFloat(float64(w2)))))
	})

	// Output:
	// X(
	//   'a' => 1.00000,
	//   'b' => 0.00000,
	//   'c' => 0.00000
	// )
}

type optionalIntSlice struct {
	A []int64
	B *[]int64
	C *map[string]int32
}

func ExampleReflector_TypeFromReflect_optionalIntSlice() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalIntSlice{}))
		px.AddTypes(c, xm)
		xm.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => Array[Integer],
	//     'b' => Optional[Array[Integer]],
	//     'c' => Optional[Hash[String, Integer[-2147483648, 2147483647]]]
	//   }
	// }]
}

func ExampleReflector_TypeFromReflect_optionalIntSliceAssign() {
	pcore.Do(func(c px.Context) {
		ois := &optionalIntSlice{}
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(ois))
		px.AddTypes(c, xm)

		ois.A = []int64{1, 2, 3}
		ois.B = &[]int64{4, 5, 6}
		ois.C = &map[string]int32{`a`: 7, `b`: 8, `c`: 9}
		px.Wrap(c, ois).ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// X(
	//   'a' => [1, 2, 3],
	//   'b' => [4, 5, 6],
	//   'c' => {
	//     'a' => 7,
	//     'b' => 8,
	//     'c' => 9
	//   }
	// )
}

func ExampleReflector_TypeFromReflect_optionalIntSlicePuppetAssign() {
	pcore.Do(func(c px.Context) {
		ois := &optionalIntSlice{}
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(ois))
		px.AddTypes(c, xm)

		pv := px.New(c, xm, px.Wrap(c, map[string]interface{}{
			`a`: []int64{1, 2, 3},
			`b`: []int64{4, 5, 6},
			`c`: map[string]int32{`a`: 7, `b`: 8, `c`: 9}}))
		pv.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// X(
	//   'a' => [1, 2, 3],
	//   'b' => [4, 5, 6],
	//   'c' => {
	//     'a' => 7,
	//     'b' => 8,
	//     'c' => 9
	//   }
	// )
}

type structSlice struct {
	A []optionalIntSlice
	B *[]optionalIntSlice
	C *[]*optionalIntSlice
}

func ExampleReflector_TypeFromReflect_structSlice() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalIntSlice{}))
		ym := c.Reflector().TypeFromReflect(`Y`, nil, reflect.TypeOf(structSlice{}))
		px.AddTypes(c, xm, ym)
		ym.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'Y',
	//   attributes => {
	//     'a' => Array[X],
	//     'b' => Optional[Array[X]],
	//     'c' => Optional[Array[Optional[X]]]
	//   }
	// }]
}

func ExampleReflector_TypeFromReflect_structSliceAssign() {
	pcore.Do(func(c px.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalIntSlice{}))
		ym := c.Reflector().TypeFromReflect(`Y`, nil, reflect.TypeOf(structSlice{}))
		px.AddTypes(c, xm, ym)
		ss := px.Wrap(c, structSlice{
			A: []optionalIntSlice{{[]int64{1, 2, 3}, &[]int64{4, 5, 6}, &map[string]int32{`a`: 7, `b`: 8, `c`: 9}}},
			B: &[]optionalIntSlice{{[]int64{11, 12, 13}, &[]int64{14, 15, 16}, &map[string]int32{`a`: 17, `b`: 18, `c`: 19}}},
			C: &[]*optionalIntSlice{{[]int64{21, 22, 23}, &[]int64{24, 25, 26}, &map[string]int32{`a`: 27, `b`: 28, `c`: 29}}},
		})
		ss.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// Y(
	//   'a' => [
	//     X(
	//       'a' => [1, 2, 3],
	//       'b' => [4, 5, 6],
	//       'c' => {
	//         'a' => 7,
	//         'b' => 8,
	//         'c' => 9
	//       }
	//     )],
	//   'b' => [
	//     X(
	//       'a' => [11, 12, 13],
	//       'b' => [14, 15, 16],
	//       'c' => {
	//         'a' => 17,
	//         'b' => 18,
	//         'c' => 19
	//       }
	//     )],
	//   'c' => [
	//     X(
	//       'a' => [21, 22, 23],
	//       'b' => [24, 25, 26],
	//       'c' => {
	//         'a' => 27,
	//         'b' => 28,
	//         'c' => 29
	//       }
	//     )]
	// )
}

// For third party vendors using anonymous fields to tag structs
type anon struct {
	_ struct{} `some:"tag here"`
	A string
}

func ExampleReflector_TypeFromReflect_structAnonField() {
	pcore.Do(func(c px.Context) {
		x := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&anon{}))
		px.AddTypes(c, x)
		x.ToString(os.Stdout, px.PrettyExpanded, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => String
	//   }
	// }]
}

type nestedInterface struct {
	A []map[string]interface{} `puppet:"type=>Array[Struct[v=>Array[String]]]"`
}

func ExampleReflector_TypeFromReflect_nestedSliceToInterface() {
	pcore.Do(func(c px.Context) {
		x := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&nestedInterface{}))
		px.AddTypes(c, x)
		v := types.CoerceTo(c, "test", x, px.Wrap(c, map[string][]map[string][]string{`a`: {{`v`: {`a`, `b`}}}}))
		v.ToString(os.Stdout, px.Pretty, nil)
		fmt.Println()
	})

	// Output:
	// X(
	//   'a' => [
	//     {
	//       'v' => ['a', 'b']
	//     }]
	// )
}
