package hash

import (
	"fmt"

	"github.com/lyraproj/pcore/px"
)

type (
	// StringHash is a mutable and order preserving hash with string keys and arbitrary values. The StringHash
	// is not safe for concurrent use unless it has been frozen by calling the method Freeze().
	StringHash interface {
		px.Equality

		// AllPair calls the given function once for each key/value pair in this hash. Return
		// true if all invocations returned true. False otherwise.
		// The method returns true if the hash i empty.
		AllPair(f func(key string, value interface{}) bool) bool

		// AnyPair calls the given function once for each key/value pair in this hash. Return
		// true when an invocation returns true. False otherwise.
		// The method returns false if the hash i empty.
		AnyPair(f func(key string, value interface{}) bool) bool

		// ComputeIfAbsent will return the value associated with the given key if the present. Otherwise it will compute
		// the value using the given mapping function and associate it key.
		ComputeIfAbsent(key string, dflt func() interface{}) interface{}

		// Copy returns a shallow copy of this hash, i.e. each key and value is not cloned
		Copy() StringHash

		// Delete the entry for the given key from the hash. Returns the old value or nil if not found
		Delete(key string) (oldValue interface{})

		// EachKey calls the given consumer function once for each key in this hash
		EachKey(consumer func(key string))

		// EachPair calls the given consumer function once for each key/value pair in this hash
		EachPair(consumer func(key string, value interface{}))

		// EachValue calls the given consumer function once for each value in this hash
		EachValue(consumer func(value interface{}))

		// Empty returns true if the hash has no entries
		Empty() bool

		// Freeze prevents further changes to the hash. If the hash is already frozen this method
		// does nothing
		Freeze()

		// Frozen returns true if this instance is frozen
		Frozen() bool

		// Get returns a value from the hash or nil together with a boolean to indicate if the key was present or not
		Get(key string) (interface{}, bool)

		// GetOrDefault returns a value from the hash or the given default if no value was found
		GetOrDefault(key string, dflt interface{}) interface{}

		// Includes returns true if the hash contains the given key
		Includes(key string) bool

		// Keys returns a slice with all the keys of the hash in the order that they were first entered
		Keys() []string

		// Len returns the number of entries in the hash
		Len() int

		// Merge this hash with the other hash giving the other precedence. A new hash is returned
		Merge(other StringHash) (merged StringHash)

		// Put adds a new key/value association to the hash or replace the value of an existing association.
		// The old value associated with the key is returned together with a boolean indicating if such an
		// association was present
		Put(key string, value interface{}) (oldValue interface{}, replaced bool)

		// PutAll copies all association from other to this hash, overwriting any existing associations
		PutAll(other StringHash)

		// Values returns a slice with all the values of the hash in the order that they were first entered
		Values() []interface{}
	}

	stringEntry struct {
		key   string
		value interface{}
	}

	stringHash struct {
		entries []stringEntry
		index   map[string]int
		frozen  bool
	}

	frozenError struct {
		key string
	}
)

var EmptyStringHash StringHash = &stringHash{[]stringEntry{}, map[string]int{}, true}

func (f *frozenError) Error() string {
	return fmt.Sprintf("attempt to add, modify, or delete key '%s' in a frozen StringHash", f.key)
}

// NewStringHash returns an empty StringHash initialized with given capacity
func NewStringHash(capacity int) StringHash {
	return &stringHash{make([]stringEntry, 0, capacity), make(map[string]int, capacity), false}
}

func (h *stringHash) AllPair(f func(key string, value interface{}) bool) bool {
	for _, e := range h.entries {
		if !f(e.key, e.value) {
			return false
		}
	}
	return true
}

func (h *stringHash) AnyPair(f func(key string, value interface{}) bool) bool {
	for _, e := range h.entries {
		if f(e.key, e.value) {
			return true
		}
	}
	return false
}

func (h *stringHash) ComputeIfAbsent(key string, dflt func() interface{}) interface{} {
	if p, ok := h.index[key]; ok {
		return h.entries[p].value
	}
	if h.frozen {
		panic(frozenError{key})
	}
	value := dflt()
	h.index[key] = len(h.entries)
	h.entries = append(h.entries, stringEntry{key, value})
	return value
}

func (h *stringHash) Copy() StringHash {
	entries := make([]stringEntry, len(h.entries))
	copy(entries, h.entries)
	index := make(map[string]int, len(h.index))
	for k, v := range h.index {
		index[k] = v
	}
	return &stringHash{entries, index, false}
}

func (h *stringHash) Delete(key string) (oldValue interface{}) {
	if h.frozen {
		panic(frozenError{key})
	}
	index := h.index
	oldValue = nil
	if p, ok := index[key]; ok {
		oldValue = h.entries[p].value
		delete(h.index, key)
		for k, v := range index {
			if v > p {
				index[k] = p - 1
			}
		}
		ne := make([]stringEntry, len(h.entries)-1)
		for i, e := range h.entries {
			if i < p {
				ne[i] = e
			} else if i > p {
				ne[i-1] = e
			}
		}
		h.entries = ne
	}
	return
}

func (h *stringHash) EachKey(consumer func(key string)) {
	for _, e := range h.entries {
		consumer(e.key)
	}
}

func (h *stringHash) EachPair(consumer func(key string, value interface{})) {
	for _, e := range h.entries {
		consumer(e.key, e.value)
	}
}

func (h *stringHash) EachValue(consumer func(value interface{})) {
	for _, e := range h.entries {
		consumer(e.value)
	}
}

func (h *stringHash) Equals(other interface{}, g px.Guard) bool {
	oh, ok := other.(*stringHash)
	if !ok || len(h.entries) != len(oh.entries) {
		return false
	}

	for _, e := range h.entries {
		oi, ok := oh.index[e.key]
		if !(ok && px.Equals(e.value, oh.entries[oi].value, g)) {
			return false
		}
	}
	return true
}

func (h *stringHash) Empty() bool {
	return len(h.entries) == 0
}

func (h *stringHash) Freeze() {
	h.frozen = true
}

func (h *stringHash) Frozen() bool {
	return h.frozen
}

func (h *stringHash) Get(key string) (interface{}, bool) {
	if p, ok := h.index[key]; ok {
		return h.entries[p].value, true
	}
	return nil, false
}

func (h *stringHash) GetOrDefault(key string, dflt interface{}) interface{} {
	if p, ok := h.index[key]; ok {
		return h.entries[p].value
	}
	return dflt
}

func (h *stringHash) Includes(key string) bool {
	_, ok := h.index[key]
	return ok
}

func (h *stringHash) Keys() []string {
	keys := make([]string, len(h.entries))
	for i, e := range h.entries {
		keys[i] = e.key
	}
	return keys
}

func (h *stringHash) Merge(other StringHash) (merged StringHash) {
	merged = h.Copy()
	merged.PutAll(other)
	return
}

func (h *stringHash) Put(key string, value interface{}) (oldValue interface{}, replaced bool) {
	if h.frozen {
		panic(frozenError{key})
	}
	var p int
	if p, replaced = h.index[key]; replaced {
		e := &h.entries[p]
		oldValue = e.value
		e.value = value
	} else {
		oldValue = nil
		h.index[key] = len(h.entries)
		h.entries = append(h.entries, stringEntry{key, value})
	}
	return
}

func (h *stringHash) PutAll(other StringHash) {
	for _, e := range other.(*stringHash).entries {
		h.Put(e.key, e.value)
	}
}

func (h *stringHash) Len() int {
	return len(h.entries)
}

func (h *stringHash) Values() []interface{} {
	values := make([]interface{}, len(h.entries))
	for i, e := range h.entries {
		values[i] = e.value
	}
	return values
}
