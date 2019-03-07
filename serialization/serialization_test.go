package serialization_test

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/serialization"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/semver/semver"
)

func ExampleNewSerializer_richDataRoundtrip() {
	pcore.Do(func(ctx px.Context) {
		ver, _ := semver.NewVersion(1, 0, 0)
		v := types.WrapSemVer(ver)
		fmt.Printf("%T '%s'\n", v, v)

		dc := serialization.NewSerializer(ctx, types.SingletonHash2(`rich_data`, types.BooleanTrue))
		buf := bytes.NewBufferString(``)
		dc.Convert(v, serialization.NewJsonStreamer(buf))

		fc := serialization.NewDeserializer(ctx, px.EmptyMap)
		serialization.JsonToData(`/tmp/sample.json`, buf, fc)
		v2 := fc.Value()

		fmt.Printf("%T '%s'\n", v2, v2)
	})
	// Output:
	// *types.SemVerValue '1.0.0'
	// *types.SemVerValue '1.0.0'
}

func ExampleNewSerializer_objectRoundtrip() {
	pcore.Do(func(ctx px.Context) {
		p := px.NewParameter(`p1`, ctx.ParseType2(`Type[String]`), nil, false)
		fmt.Println(p)

		dc := serialization.NewSerializer(ctx, px.EmptyMap)
		buf := bytes.NewBufferString(``)
		dc.Convert(types.WrapValues([]px.Value{p, p}), serialization.NewJsonStreamer(buf))

		fc := serialization.NewDeserializer(ctx, px.EmptyMap)
		b := buf.String()
		fmt.Println(b)
		serialization.JsonToData(`/tmp/sample.json`, buf, fc)
		p2 := fc.Value().(px.List).At(0)

		fmt.Println(p2)
	})
	// Output:
	// Parameter('name' => 'p1', 'type' => Type[String])
	// [{"__ptype":"Parameter","name":"p1","type":{"__ptype":"Type","__pvalue":"Type[String]"}},{"__pref":1}]
	// Parameter('name' => 'p1', 'type' => Type[String])
}

func ExampleNewSerializer_structInArrayRoundtrip() {
	pcore.Do(func(ctx px.Context) {
		p := types.WrapValues([]px.Value{ctx.ParseType2(`Struct[a => String, b => Integer]`)})
		fmt.Println(p)
		dc := serialization.NewSerializer(ctx, px.EmptyMap)
		buf := bytes.NewBufferString(``)
		dc.Convert(p, serialization.NewJsonStreamer(buf))

		fc := serialization.NewDeserializer(ctx, px.EmptyMap)
		b := buf.String()
		fmt.Println(b)
		serialization.JsonToData(`/tmp/sample.json`, buf, fc)
		p2 := fc.Value()

		fmt.Println(p2)
	})
	// Output:
	// [Struct[{'a' => String, 'b' => Integer}]]
	// [{"__ptype":"Type","__pvalue":"Struct[{'a' =\u003e String, 'b' =\u003e Integer}]"}]
	// [Struct[{'a' => String, 'b' => Integer}]]
}

func ExampleNewSerializer_typeSetRoundtrip() {
	pcore.Do(func(ctx px.Context) {
		p := ctx.ParseType2(`TypeSet[{
      name => 'Foo',
      version => '1.0.0',
      pcore_version => '1.0.0',
      types => {
        Bar => Object[
  attributes => {
    subnet_id => { type => Optional[String], value => 'FAKED_SUBNET_ID' },
    vpc_id => String,
    cidr_block => String,
    map_public_ip_on_launch => Boolean
  }
        ]
      }}]`)
		px.AddTypes(ctx, p)
		fmt.Println(p)
		dc := serialization.NewSerializer(pcore.RootContext(), px.EmptyMap)
		buf := bytes.NewBufferString(``)
		dc.Convert(p, serialization.NewJsonStreamer(buf))

		fc := serialization.NewDeserializer(ctx, px.EmptyMap)
		b := buf.String()
		fmt.Println(b)
		serialization.JsonToData(`/tmp/sample.json`, buf, fc)
		p2 := fc.Value()
		fmt.Println(p2)
	})
	// Output:
	// TypeSet[{pcore_version => '1.0.0', name_authority => 'http://puppet.com/2016.1/runtime', name => 'Foo', version => '1.0.0', types => {Bar => {attributes => {'subnet_id' => {'type' => Optional[String], 'value' => 'FAKED_SUBNET_ID'}, 'vpc_id' => String, 'cidr_block' => String, 'map_public_ip_on_launch' => Boolean}}}}]
	// {"__ptype":"Pcore::TypeSet","pcore_version":{"__ptype":"SemVer","__pvalue":"1.0.0"},"name_authority":{"__ptype":"URI","__pvalue":"http://puppet.com/2016.1/runtime"},"name":"Foo","version":{"__ptype":"SemVer","__pvalue":"1.0.0"},"types":{"Bar":{"__ptype":"Pcore::ObjectType","name":"Foo::Bar","attributes":{"subnet_id":{"type":{"__ptype":"Type","__pvalue":"Optional[String]"},"value":"FAKED_SUBNET_ID"},"vpc_id":{"__ptype":"Type","__pvalue":"String"},"cidr_block":{"__pref":44},"map_public_ip_on_launch":{"__ptype":"Type","__pvalue":"Boolean"}}}}}
	// TypeSet[{pcore_version => '1.0.0', name_authority => 'http://puppet.com/2016.1/runtime', name => 'Foo', version => '1.0.0', types => {Bar => {attributes => {'subnet_id' => {'type' => Optional[String], 'value' => 'FAKED_SUBNET_ID'}, 'vpc_id' => String, 'cidr_block' => String, 'map_public_ip_on_launch' => Boolean}}}}]
}

func ExampleNewSerializer_goValueRoundtrip() {
	type MyInt int

	pcore.Do(func(ctx px.Context) {
		mi := MyInt(32)
		px.AddTypes(ctx, ctx.Reflector().TypeFromReflect(`Test::MyInt`, nil, reflect.TypeOf(mi)))

		v := px.Wrap(ctx, mi)
		fmt.Println(v)

		dc := serialization.NewSerializer(pcore.RootContext(), px.EmptyMap)
		buf := bytes.NewBufferString(``)
		dc.Convert(v, serialization.NewJsonStreamer(buf))

		fc := serialization.NewDeserializer(ctx, px.EmptyMap)
		serialization.JsonToData(`/tmp/sample.json`, buf, fc)
		v2 := fc.Value()

		fmt.Println(v2)
	})
	// Output:
	// Test::MyInt('value' => 32)
	// Test::MyInt('value' => 32)
}

func ExampleNewSerializer_goStructRoundtrip() {
	type MyStruct struct {
		X int
		Y string
	}

	pcore.Do(func(ctx px.Context) {
		mi := &MyStruct{32, "hello"}
		px.AddTypes(ctx, ctx.Reflector().TypeFromReflect(`Test::MyStruct`, nil, reflect.TypeOf(mi)))

		v := px.Wrap(ctx, mi)
		fmt.Println(v)

		dc := serialization.NewSerializer(pcore.RootContext(), px.EmptyMap)
		buf := bytes.NewBufferString(``)
		dc.Convert(v, serialization.NewJsonStreamer(buf))

		fc := serialization.NewDeserializer(ctx, px.EmptyMap)
		serialization.JsonToData(`/tmp/sample.json`, buf, fc)
		v2 := fc.Value()

		fmt.Println(v2)
		ms2 := v2.(px.Reflected).Reflect(ctx).Interface()
		fmt.Printf("%T %v\n", ms2, ms2)
	})
	// Output:
	// Test::MyStruct('x' => 32, 'y' => 'hello')
	// Test::MyStruct('x' => 32, 'y' => 'hello')
	// serialization_test.MyStruct {32 hello}
}

func ExampleNewSerializer_goStructWithDynamicRoundtrip() {
	type MyStruct struct {
		X px.List
		Y px.OrderedMap
	}

	pcore.Do(func(ctx px.Context) {
		mi := &MyStruct{px.Wrap(ctx, []int{32}).(px.List), px.Wrap(ctx, map[string]string{"msg": "hello"}).(px.OrderedMap)}
		px.AddTypes(ctx, ctx.Reflector().TypeFromReflect(`Test::MyStruct`, nil, reflect.TypeOf(mi)))

		v := px.Wrap(ctx, mi)
		fmt.Println(v)

		dc := serialization.NewSerializer(pcore.RootContext(), px.EmptyMap)
		buf := bytes.NewBufferString(``)
		dc.Convert(v, serialization.NewJsonStreamer(buf))

		fc := serialization.NewDeserializer(ctx, px.EmptyMap)
		serialization.JsonToData(`/tmp/sample.json`, buf, fc)
		v2 := fc.Value()

		fmt.Println(v2)
		ms2 := v2.(px.Reflected).Reflect(ctx).Interface()
		fmt.Printf("%T %v\n", ms2, ms2)
	})
	// Output:
	// Test::MyStruct('x' => [32], 'y' => {'msg' => 'hello'})
	// Test::MyStruct('x' => [32], 'y' => {'msg' => 'hello'})
	// serialization_test.MyStruct {[32] {'msg' => 'hello'}}
}

func ExampleSerializer_Convert() {
	pcore.Do(func(ctx px.Context) {
		ver, _ := semver.NewVersion(1, 0, 0)
		cl := px.NewCollector()
		serialization.NewSerializer(ctx, types.SingletonHash2(`rich_data`, types.BooleanTrue)).Convert(types.WrapSemVer(ver), cl)
		fmt.Println(cl.Value())
	})
	// Output: {'__ptype' => 'SemVer', '__pvalue' => '1.0.0'}
}

func ExampleNewJsonStreamer() {
	pcore.Do(func(ctx px.Context) {
		buf := bytes.NewBufferString(``)
		serialization.NewSerializer(ctx, px.EmptyMap).Convert(
			types.WrapStringToInterfaceMap(ctx, map[string]interface{}{`__ptype`: `SemVer`, `__pvalue`: `1.0.0`}), serialization.NewJsonStreamer(buf))
		fmt.Println(buf)
	})
	// Output: {"__ptype":"SemVer","__pvalue":"1.0.0"}
}

func ExampleJsonToData() {
	pcore.Do(func(ctx px.Context) {
		buf := bytes.NewBufferString(`{"__ptype":"SemVer","__pvalue":"1.0.0"}`)
		fc := px.NewCollector()
		serialization.JsonToData(`/tmp/ver.json`, buf, fc)
		fmt.Println(fc.Value())
	})
	// Output: {'__ptype' => 'SemVer', '__pvalue' => '1.0.0'}
}
