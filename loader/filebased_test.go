package loader_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
)

func TestFileBasedAlias(t *testing.T) {
	pcore.Do(func(c px.Context) {
		c.DoWithLoader(px.NewFileBasedLoader(c.Loader(), `testdata`, ``, px.PuppetDataTypePath), func() {
			v, ok := px.Load(c, px.NewTypedName(px.NsType, `MyType`))
			require.True(t, ok, `failed to load type`)
			tp, ok := v.(px.Type)
			require.True(t, ok, `loaded element is not a type`)
			require.Equal(t, `MyType`, tp.Name())
		})
	})
}

func TestFileBased_parseError(t *testing.T) {
	pcore.Do(func(c px.Context) {
		c.DoWithLoader(px.NewFileBasedLoader(c.Loader(), `testdata`, ``, px.PuppetDataTypePath), func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(error); ok {
						require.Equal(t, `expected one of ',' or '}', got 'third' (file: testdata/types/badtype.pp, line: 5, column: 5)`, err.Error())
					} else {
						panic(r)
					}
				}
			}()
			px.Load(c, px.NewTypedName(px.NsType, `BadType`))
			require.Fail(t, `expected panic didn't happen`)
		})
	})
}
