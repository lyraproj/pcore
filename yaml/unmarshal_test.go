package yaml_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/yaml"
)

const text = `foo:
  bar: [1, 2, {hello: sub}]
bar:
  - 1
  - 2
`

func TestUnmarshal(t *testing.T) {
	pcore.Do(func(c px.Context) {
		v := yaml.Unmarshal(c, []byte(text))
		require.Equal(t, `{'foo' => {'bar' => [1, 2, {'hello' => 'sub'}]}, 'bar' => [1, 2]}`, v.String())
	})
}

func TestUnmarshalWithPositions(t *testing.T) {
	pcore.Do(func(c px.Context) {
		v := yaml.UnmarshalWithPositions(c, []byte(text))
		require.Equal(t, `{'foo' => {'bar' => [1, 2, {'hello' => 'sub'}]}, 'bar' => [1, 2]}`, v.String())
		bar, ok := v.Value.(px.OrderedMap).Get4(`bar`)
		require.True(t, ok)
		barPos, ok := bar.(*yaml.Value)
		require.True(t, ok)
		require.Equal(t, 4, barPos.Line)
		require.Equal(t, 3, barPos.Column)
	})
}
