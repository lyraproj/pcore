package loader_test

import (
	"testing"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
	"github.com/stretchr/testify/assert"
)

func TestFileBasedAlias(t *testing.T) {
	pcore.Do(func(c px.Context) {
		c.DoWithLoader(px.NewFileBasedLoader(c.Loader(), `testdata`, ``, px.PuppetDataTypePath), func() {
			v, ok := px.Load(c, px.NewTypedName(px.NsType, `MyType`))
			assert.True(t, ok, `failed to load type`)
			tp, ok := v.(px.Type)
			assert.True(t, ok, `loaded element is not a type`)
			assert.Equal(t, `MyType`, tp.Name())
		})
	})
}
