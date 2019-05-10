package types_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/lyraproj/pcore/serialization"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/semver/semver"
	"github.com/stretchr/testify/require"
)

type ManifestService struct {
}

func (m *ManifestService) Invoke(identifier, name string, arguments ...px.Value) px.Value {
	return nil
}

func (m *ManifestService) Metadata() (px.TypeSet, []Definition) {
	return nil, nil
}

func (m *ManifestService) State(name string, parameters px.OrderedMap) px.PuppetObject {
	return nil
}

type Definition interface {
	px.Value
	issue.Labeled

	// Identifier returns a TypedName that uniquely identifies the step within the service.
	Identifier() px.TypedName

	// ServiceId is the identifier of the service
	ServiceId() px.TypedName

	// Properties is an ordered map of properties of this definition. Will be of type
	// Hash[Pattern[/\A[a-z][A-Za-z]+\z/],RichData]
	Properties() px.OrderedMap
}

func TestCallable_ToString(t *testing.T) {
	px.NewGoObjectType(`Service::Definition`, reflect.TypeOf((*Definition)(nil)).Elem(), `{
    attributes => {
      identifier => TypedName,
      serviceId => TypedName,
      properties => Hash[String,RichData]
    }
  }`)

	var mt px.TypeSet
	pcore.Do(func(r px.Context) {
		pcore.DoWithParent(r, func(c px.Context) {
			mt = c.Reflector().TypeSetFromReflect(`Test`, semver.MustParseVersion(`1.0.0`), nil, reflect.TypeOf(&ManifestService{}))
			px.AddTypes(c, mt)
			s := bytes.NewBufferString(``)
			mt.ToString(s, px.PrettyExpanded, nil)
			ts := s.String()
			mt2 := c.ParseType(ts)
			px.AddTypes(c, mt2)
			require.True(t, mt.Equals(mt2, nil))
		})

		buf := bytes.NewBufferString(``)
		pcore.DoWithParent(r, func(c px.Context) {
			dc := serialization.NewSerializer(c, px.EmptyMap)
			dc.Convert(mt, serialization.NewJsonStreamer(buf))
		})

		pcore.DoWithParent(r, func(c px.Context) {
			fc := serialization.NewDeserializer(c, px.EmptyMap)
			serialization.JsonToData(`/tmp/sample.json`, buf, fc)
			mtd := fc.Value()
			s := bytes.NewBufferString(``)
			mtd.ToString(s, px.PrettyExpanded, nil)
			fmt.Println(s)
			ts := s.String()
			mt2 := c.ParseType(ts)
			px.AddTypes(c, mt2)
			require.True(t, mt.Equals(mtd, nil))
		})
	})
}
