package types

import (
	"reflect"
	"testing"
)

func Test_isNilAndUnknown(t *testing.T) {
	rt := reflect.TypeOf((*interface{})(nil)).Elem()
	if !isNilAndUnknown(rt) {
		t.Fatal(`type of interface{} is not nil and unknown`)
	}
}
