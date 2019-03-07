package types

import (
	"reflect"

	"github.com/lyraproj/pcore/px"
)

type taggedType struct {
	typ         reflect.Type
	puppetTags  map[string]string
	annotations px.OrderedMap
}

func init() {
	px.NewTaggedType = func(typ reflect.Type, puppetTags map[string]string) px.AnnotatedType {
		return &taggedType{typ, puppetTags, emptyMap}
	}

	px.NewAnnotatedType = func(typ reflect.Type, puppetTags map[string]string, annotations px.OrderedMap) px.AnnotatedType {
		return &taggedType{typ, puppetTags, annotations}
	}
}

func (tg *taggedType) Annotations() px.OrderedMap {
	return tg.annotations
}

func (tg *taggedType) Type() reflect.Type {
	return tg.typ
}

func (tg *taggedType) Tags() map[string]px.OrderedMap {
	fs := Fields(tg.typ)
	nf := len(fs)
	tags := make(map[string]px.OrderedMap, 7)
	if nf > 0 {
		for i, f := range fs {
			if i == 0 && f.Anonymous {
				// Parent
				continue
			}
			if f.PkgPath != `` {
				// Unexported
				continue
			}
			if ft, ok := TagHash(&f); ok {
				tags[f.Name] = ft
			}
		}
	}
	if tg.puppetTags != nil && len(tg.puppetTags) > 0 {
		for k, v := range tg.puppetTags {
			if h, ok := ParseTagHash(v); ok {
				tags[k] = h
			}
		}
	}
	return tags
}
