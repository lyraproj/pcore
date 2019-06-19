package internal_test

import (
	"testing"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
)

func TestPcore(t *testing.T) {
	pcore.Do(func(ctx px.Context) {
		l, _ := px.Load(ctx, px.NewTypedName(px.NsType, `Pcore::ObjectTypeExtensionType`))
		x, ok := l.(px.Type)
		if !(ok && x.Name() == `Pcore::ObjectTypeExtensionType`) {
			t.Errorf(`failed to load %s`, `Pcore::ObjectTypeExtensionType`)
		}
	})
}
