package types

import (
	"io"

	"github.com/lyraproj/pcore/utils"

	"github.com/lyraproj/pcore/errors"
	"github.com/lyraproj/pcore/px"
)

type (
	IteratorType struct {
		typ px.Type
	}

	iteratorValue struct {
		iterator px.Iterator
	}

	indexedIterator struct {
		elementType px.Type
		pos         int
		indexed     px.List
	}

	mappingIterator struct {
		elementType px.Type
		mapFunc     px.Mapper
		base        px.Iterator
	}

	predicateIterator struct {
		predicate px.Predicate
		outcome   bool
		base      px.Iterator
	}
)

var iteratorTypeDefault = &IteratorType{typ: DefaultAnyType()}

var IteratorMetaType px.ObjectType

func init() {
	IteratorMetaType = newObjectType(`Pcore::IteratorType`,
		`Pcore::AnyType {
			attributes => {
				type => {
					type => Optional[Type],
					value => Any
				},
			}
		}`, func(ctx px.Context, args []px.Value) px.Value {
			return newIteratorType2(args...)
		})
}

func DefaultIteratorType() *IteratorType {
	return iteratorTypeDefault
}

func NewIteratorType(elementType px.Type) *IteratorType {
	if elementType == nil || elementType == anyTypeDefault {
		return DefaultIteratorType()
	}
	return &IteratorType{elementType}
}

func newIteratorType2(args ...px.Value) *IteratorType {
	switch len(args) {
	case 0:
		return DefaultIteratorType()
	case 1:
		containedType, ok := args[0].(px.Type)
		if !ok {
			panic(NewIllegalArgumentType(`Iterator[]`, 0, `Type`, args[0]))
		}
		return NewIteratorType(containedType)
	default:
		panic(errors.NewIllegalArgumentCount(`Iterator[]`, `0 - 1`, len(args)))
	}
}

func (t *IteratorType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *IteratorType) Default() px.Type {
	return iteratorTypeDefault
}

func (t *IteratorType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*IteratorType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *IteratorType) Generic() px.Type {
	return NewIteratorType(px.GenericType(t.typ))
}

func (t *IteratorType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *IteratorType) IsAssignable(o px.Type, g px.Guard) bool {
	if it, ok := o.(*IteratorType); ok {
		return GuardedIsAssignable(t.typ, it.typ, g)
	}
	return false
}

func (t *IteratorType) IsInstance(o px.Value, g px.Guard) bool {
	if it, ok := o.(px.Iterator); ok {
		return GuardedIsInstance(t.typ, it.ElementType(), g)
	}
	return false
}

func (t *IteratorType) MetaType() px.ObjectType {
	return IteratorMetaType
}

func (t *IteratorType) Name() string {
	return `Iterator`
}

func (t *IteratorType) Parameters() []px.Value {
	if t.typ == DefaultAnyType() {
		return px.EmptyValues
	}
	return []px.Value{t.typ}
}

func (t *IteratorType) CanSerializeAsString() bool {
	return canSerializeAsString(t.typ)
}

func (t *IteratorType) SerializationString() string {
	return t.String()
}

func (t *IteratorType) String() string {
	return px.ToString2(t, None)
}

func (t *IteratorType) ElementType() px.Type {
	return t.typ
}

func (t *IteratorType) Resolve(c px.Context) px.Type {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *IteratorType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IteratorType) PType() px.Type {
	return &TypeType{t}
}

func WrapIterator(iter px.Iterator) px.IteratorValue {
	return &iteratorValue{iter}
}

func (it *iteratorValue) AsArray() px.List {
	return it.iterator.AsArray()
}

func (it *iteratorValue) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*iteratorValue); ok {
		return it.iterator.ElementType().Equals(ot.iterator.ElementType(), g)
	}
	return false
}

func (it *iteratorValue) PType() px.Type {
	return NewIteratorType(it.iterator.ElementType())
}

func (it *iteratorValue) String() string {
	return px.ToString2(it, None)
}

func (it *iteratorValue) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	if it.iterator.ElementType() != DefaultAnyType() {
		utils.WriteString(b, `Iterator[`)
		px.GenericType(it.iterator.ElementType()).ToString(b, s, g)
		utils.WriteString(b, `]-Value`)
	} else {
		utils.WriteString(b, `Iterator-Value`)
	}
}

func stopIteration() {
	if err := recover(); err != nil {
		if _, ok := err.(*errors.StopIteration); !ok {
			panic(err)
		}
	}
}

func find(iter px.Iterator, predicate px.Predicate, dflt px.Value, dfltProducer px.Producer) (result px.Value) {
	defer stopIteration()

	result = px.Undef
	var ok bool
	for {
		result, ok = iter.Next()
		if !ok {
			if dfltProducer != nil {
				result = dfltProducer()
			} else {
				result = dflt
			}
			break
		}
		if predicate(result) {
			break
		}
	}
	return
}

func each(iter px.Iterator, consumer px.Consumer) {
	defer stopIteration()

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		consumer(v)
	}
}

func eachWithIndex(iter px.Iterator, consumer px.BiConsumer) {
	defer stopIteration()

	for idx := int64(0); ; idx++ {
		v, ok := iter.Next()
		if !ok {
			break
		}
		consumer(integerValue(idx), v)
	}
}

func all(iter px.Iterator, predicate px.Predicate) (result bool) {
	defer stopIteration()

	result = true
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if !predicate(v) {
			result = false
			break
		}
	}
	return
}

func any(iter px.Iterator, predicate px.Predicate) (result bool) {
	defer stopIteration()

	result = false
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if predicate(v) {
			result = true
			break
		}
	}
	return
}

func reduce2(iter px.Iterator, value px.Value, redactor px.BiMapper) (result px.Value) {
	defer stopIteration()

	result = value
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		result = redactor(result, v)
	}
	return
}

func reduce(iter px.Iterator, redactor px.BiMapper) px.Value {
	v, ok := iter.Next()
	if !ok {
		return undef
	}
	return reduce2(iter, v, redactor)
}

func asArray(iter px.Iterator) (result px.List) {
	el := make([]px.Value, 0, 16)
	defer func() {
		if err := recover(); err != nil {
			if _, ok := err.(*errors.StopIteration); ok {
				result = WrapValues(el)
			} else {
				panic(err)
			}
		}
	}()

	for {
		v, ok := iter.Next()
		if !ok {
			result = WrapValues(el)
			break
		}
		if it, ok := v.(px.IteratorValue); ok {
			v = it.AsArray()
		}
		el = append(el, v)
	}
	return
}

func (ai *indexedIterator) All(predicate px.Predicate) bool {
	return all(ai, predicate)
}

func (ai *indexedIterator) Any(predicate px.Predicate) bool {
	return any(ai, predicate)
}

func (ai *indexedIterator) Each(consumer px.Consumer) {
	each(ai, consumer)
}

func (ai *indexedIterator) EachWithIndex(consumer px.BiConsumer) {
	eachWithIndex(ai, consumer)
}

func (ai *indexedIterator) ElementType() px.Type {
	return ai.elementType
}

func (ai *indexedIterator) Find(predicate px.Predicate) px.Value {
	return find(ai, predicate, undef, nil)
}

func (ai *indexedIterator) Find2(predicate px.Predicate, dflt px.Value) px.Value {
	return find(ai, predicate, dflt, nil)
}

func (ai *indexedIterator) Find3(predicate px.Predicate, dflt px.Producer) px.Value {
	return find(ai, predicate, nil, dflt)
}

func (ai *indexedIterator) Next() (px.Value, bool) {
	pos := ai.pos + 1
	if pos < ai.indexed.Len() {
		ai.pos = pos
		return ai.indexed.At(pos), true
	}
	return undef, false
}

func (ai *indexedIterator) Map(elementType px.Type, mapFunc px.Mapper) px.IteratorValue {
	return WrapIterator(&mappingIterator{elementType, mapFunc, ai})
}

func (ai *indexedIterator) Reduce(redactor px.BiMapper) px.Value {
	return reduce(ai, redactor)
}

func (ai *indexedIterator) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return reduce2(ai, initialValue, redactor)
}

func (ai *indexedIterator) Reject(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, false, ai})
}

func (ai *indexedIterator) Select(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, true, ai})
}

func (ai *indexedIterator) AsArray() px.List {
	return ai.indexed
}

func (ai *predicateIterator) All(predicate px.Predicate) bool {
	return all(ai, predicate)
}

func (ai *predicateIterator) Any(predicate px.Predicate) bool {
	return any(ai, predicate)
}

func (ai *predicateIterator) Next() (v px.Value, ok bool) {
	defer func() {
		if err := recover(); err != nil {
			if _, ok = err.(*errors.StopIteration); ok {
				ok = false
				v = undef
			} else {
				panic(err)
			}
		}
	}()

	for {
		v, ok = ai.base.Next()
		if !ok {
			v = undef
			break
		}
		if ai.predicate(v) == ai.outcome {
			break
		}
	}
	return
}

func (ai *predicateIterator) Each(consumer px.Consumer) {
	each(ai, consumer)
}

func (ai *predicateIterator) EachWithIndex(consumer px.BiConsumer) {
	eachWithIndex(ai, consumer)
}

func (ai *predicateIterator) ElementType() px.Type {
	return ai.base.ElementType()
}

func (ai *predicateIterator) Find(predicate px.Predicate) px.Value {
	return find(ai, predicate, undef, nil)
}

func (ai *predicateIterator) Find2(predicate px.Predicate, dflt px.Value) px.Value {
	return find(ai, predicate, dflt, nil)
}

func (ai *predicateIterator) Find3(predicate px.Predicate, dflt px.Producer) px.Value {
	return find(ai, predicate, nil, dflt)
}

func (ai *predicateIterator) Map(elementType px.Type, mapFunc px.Mapper) px.IteratorValue {
	return WrapIterator(&mappingIterator{elementType, mapFunc, ai})
}

func (ai *predicateIterator) Reduce(redactor px.BiMapper) px.Value {
	return reduce(ai, redactor)
}

func (ai *predicateIterator) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return reduce2(ai, initialValue, redactor)
}

func (ai *predicateIterator) Reject(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, false, ai})
}

func (ai *predicateIterator) Select(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, true, ai})
}

func (ai *predicateIterator) AsArray() px.List {
	return asArray(ai)
}

func (ai *mappingIterator) All(predicate px.Predicate) bool {
	return all(ai, predicate)
}

func (ai *mappingIterator) Any(predicate px.Predicate) bool {
	return any(ai, predicate)
}

func (ai *mappingIterator) Next() (v px.Value, ok bool) {
	v, ok = ai.base.Next()
	if !ok {
		v = undef
	} else {
		v = ai.mapFunc(v)
	}
	return
}

func (ai *mappingIterator) Each(consumer px.Consumer) {
	each(ai, consumer)
}

func (ai *mappingIterator) EachWithIndex(consumer px.BiConsumer) {
	eachWithIndex(ai, consumer)
}

func (ai *mappingIterator) ElementType() px.Type {
	return ai.elementType
}

func (ai *mappingIterator) Find(predicate px.Predicate) px.Value {
	return find(ai, predicate, undef, nil)
}

func (ai *mappingIterator) Find2(predicate px.Predicate, dflt px.Value) px.Value {
	return find(ai, predicate, dflt, nil)
}

func (ai *mappingIterator) Find3(predicate px.Predicate, dflt px.Producer) px.Value {
	return find(ai, predicate, nil, dflt)
}

func (ai *mappingIterator) Map(elementType px.Type, mapFunc px.Mapper) px.IteratorValue {
	return WrapIterator(&mappingIterator{elementType, mapFunc, ai})
}

func (ai *mappingIterator) Reduce(redactor px.BiMapper) px.Value {
	return reduce(ai, redactor)
}

func (ai *mappingIterator) Reduce2(initialValue px.Value, redactor px.BiMapper) px.Value {
	return reduce2(ai, initialValue, redactor)
}

func (ai *mappingIterator) Reject(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, false, ai})
}

func (ai *mappingIterator) Select(predicate px.Predicate) px.IteratorValue {
	return WrapIterator(&predicateIterator{predicate, true, ai})
}

func (ai *mappingIterator) AsArray() px.List {
	return asArray(ai)
}
