package loader_test

import (
	"github.com/lyraproj/pcore/eval"
	"github.com/stretchr/testify/assert"
	"testing"

	// Initialize pcore
	_ "github.com/lyraproj/pcore/pcore"
)

func TestFileBasedAlias(t *testing.T) {
	eval.Puppet.Do(func(c eval.Context) {
		c.DoWithLoader(eval.NewFileBasedLoader(c.Loader(), `testdata`, ``, eval.PuppetDataTypePath), func() {
			v, ok := eval.Load(c, eval.NewTypedName(eval.NsType, `MyType`))
			assert.True(t, ok, `failed to load type`)
			tp, ok := v.(eval.Type)
			assert.True(t, ok, `loaded element is not a type`)
			assert.Equal(t, `MyType`, tp.Name())
		})
	})
}
