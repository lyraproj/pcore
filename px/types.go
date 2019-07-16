package px

import (
	"reflect"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/semver/semver"
)

type (
	Visitor func(t Type)

	Type interface {
		Value

		IsInstance(o Value, g Guard) bool

		IsAssignable(t Type, g Guard) bool

		MetaType() ObjectType

		Name() string

		Accept(visitor Visitor, g Guard)
	}

	SizedType interface {
		Type

		Size() Type
	}

	StringType interface {
		SizedType

		Value() *string
	}

	Creatable interface {
		Constructor(c Context) Function
	}

	Newable interface {
		New(c Context, args []Value) Value
	}

	ResolvableType interface {
		Name() string

		Resolve(c Context) Type
	}

	ParameterizedType interface {
		Type

		Default() Type

		// Parameters returns the parameters that is needed in order to recreate
		// an instance of the parameterized type.
		Parameters() []Value
	}

	SerializeAsString interface {
		// CanSerializeAsString responds true if this instance and all its nested
		// instances can serialize as string
		CanSerializeAsString() bool

		// SerializationString returns the string that the type of the instance can use
		// to recreate the instance
		SerializationString() string
	}

	Annotation interface {
		Validate(Context, Annotatable)
	}

	Annotatable interface {
		Annotations(c Context) OrderedMap
	}

	CallableMember interface {
		Call(c Context, receiver Value, block Lambda, args []Value) Value
	}

	CallableGoMember interface {
		// CallGo calls a member on a struct pointer with the given arguments
		CallGo(c Context, receiver interface{}, args ...interface{}) []interface{}

		// CallGoReflected is like Call but using reflected arguments and return value. The
		// first argument is the receiver
		CallGoReflected(c Context, args []reflect.Value) []reflect.Value
	}

	TypeWithCallableMembers interface {
		// Member returns an attribute reader or other function and true, or nil and false if no such member exists
		Member(name string) (CallableMember, bool)
	}

	AnnotatedMember interface {
		Annotatable
		Equality
		CallableMember

		Name() string

		Label() string

		FeatureType() string

		Container() ObjectType

		Type() Type

		Override() bool

		Final() bool

		InitHash() OrderedMap

		Accept(v Visitor, g Guard)

		CallableType() Type
	}

	AttributeKind string

	TagsAnnotation interface {
		PuppetObject

		Tag(key string) string

		Tags() OrderedMap
	}

	Attribute interface {
		AnnotatedMember
		Kind() AttributeKind

		// Get returns this attributes value in the given instance
		Get(instance Value) Value

		// HasValue returns true if a value has been defined for this attribute.
		HasValue() bool

		// Default returns true if the given value equals the default value for this attribute
		Default(value Value) bool

		// Value returns the value of this attribute, or raises an error if no value has been defined.
		Value() Value

		// GoName Returns the name of the struct field that this attribute maps to when applicable or
		// an empty string.
		GoName() string

		// Tags returns the TagAnnotation for this attribute or nil if the attribute has no tags.
		Tags(Context) TagsAnnotation
	}

	ObjFunc interface {
		AnnotatedMember

		// GoName Returns the name of the struct field that this attribute maps to when applicable or
		// an empty string.
		GoName() string

		// ReturnsError returns true if the underlying method returns an error instance in case of
		// failure. Such errors must be converted to panics by the caller
		ReturnsError() bool

		// TakesContext returns true if the first argument of the actual Go function
		// is a context.Context
		TakesContext() bool
	}

	AttributesInfo interface {
		NameToPos() map[string]int

		Attributes() []Attribute

		EqualityAttributeIndex() []int

		RequiredCount() int

		PositionalFromHash(hash OrderedMap) []Value
	}

	ObjectType interface {
		Annotatable
		ParameterizedType
		TypeWithCallableMembers
		Creatable

		HasHashConstructor() bool

		Functions(includeParent bool) []ObjFunc

		// Returns the Go reflect.Type that this type was reflected from, if any.
		//
		GoType() reflect.Type

		// InstanceHash returns the InitHash for the given instance. The instance must be of this type
		InstanceHash(o Value) OrderedMap

		// IsInterface returns true for non parameterized types that contains only methods
		IsInterface() bool

		IsMetaType() bool

		IsParameterized() bool

		// Implements returns true the receiver implements all methods of ObjectType
		Implements(ObjectType, Guard) bool

		AttributesInfo() AttributesInfo

		// FromReflectedValue creates a new instance of the receiver type
		// and initializes that instance from the given src
		FromReflectedValue(c Context, src reflect.Value) PuppetObject

		// Parent returns the type that this type inherits from or nil if
		// the type doesn't have a parent
		Parent() Type

		// ToReflectedValue copies values from src to dest. The src argument
		// must be an instance of the receiver. The dest argument must be
		// a reflected struct. The src must be able to deliver a value to
		// each of the exported fields in dest.
		//
		// Puppets name convention stipulates lower case names using
		// underscores to separate words. The Go conversion is to use
		// camel cased names. ReflectValueTo will convert camel cased names
		// into names with underscores.
		ToReflectedValue(c Context, src PuppetObject, dest reflect.Value)
	}

	TypeSet interface {
		ParameterizedType

		// GetType returns the given type from the receiver together with
		// a flag indicating success or failure
		GetType(typedName TypedName) (Type, bool)

		// GetType2 is like GetType but uses a string to identify the type
		GetType2(name string) (Type, bool)

		// Authority returns the name authority of the receiver
		NameAuthority() URI

		// TypedName returns the name of this type set as a TypedName
		TypedName() TypedName

		// Types returns a hash of all types contained in this set. The keys
		// in this hash are relative to the receiver name
		Types() OrderedMap

		// Version returns the version of the receiver
		Version() semver.Version
	}

	TypeWithContainedType interface {
		Type

		ContainedType() Type
	}

	// Generalizable implemented by all parameterized types that have type parameters
	Generalizable interface {
		ParameterizedType
		Generic() Type
	}
)

var CommonType func(a Type, b Type) Type

var GenericType func(t Type) Type

var IsInstance func(puppetType Type, value Value) bool

// IsAssignable answers if t is assignable to this type
var IsAssignable func(puppetType Type, other Type) bool

var Generalize func(t Type) Type

var Normalize func(t Type) Type

var DefaultFor func(t Type) Type

func AssertType(pfx interface{}, expected, actual Type) Type {
	if !IsAssignable(expected, actual) {
		panic(TypeMismatchError(pfx, expected, actual))
	}
	return actual
}

func AssertInstance(pfx interface{}, expected Type, value Value) Value {
	if !IsInstance(expected, value) {
		panic(MismatchError(pfx, expected, value))
	}
	return value
}

func MismatchError(pfx interface{}, expected Type, value Value) issue.Reported {
	return Error(TypeMismatch, issue.H{`detail`: DescribeMismatch(getPrefix(pfx), expected, DetailedValueType(value))})
}

func TypeMismatchError(pfx interface{}, expected Type, actual Type) issue.Reported {
	return Error(TypeMismatch, issue.H{`detail`: DescribeMismatch(getPrefix(pfx), expected, actual)})
}

// New creates a new instance of type t
var New func(c Context, receiver Value, args ...Value) Value

// New creates a new instance of type t and calls the block with the created instance. It
// returns the value returned from the block
func NewWithBlock(c Context, receiver Value, args []Value, block Lambda) Value {
	r := New(c, receiver, args...)
	if block != nil {
		r = block.Call(c, nil, r)
	}
	return r
}

var DescribeSignatures func(signatures []Signature, argsTuple Type, block Lambda) string

// DescribeMismatch returns a string that describes a mismatch between the expected and actual Type. The
// string is prefixed with the given pfx
var DescribeMismatch func(pfx string, expected Type, actual Type) string

// NewGoType will infer the Pcore type from the public attributes and functions of the provided
// zeroValue which must be a struct or a pointer to a struct.
var NewGoType func(name string, zeroValue interface{}) ObjectType

// NewGoObjectType is like NewObjectType but it is intended for Go types that do not implement the
// PuppetObject interface. Instead, the reflect.Type of the null value of a go instance is passed and later
// registered with the ImplementationRegistry. This helps Pcore to correctly compute the pcore Type of the go
// instance.
var NewGoObjectType func(name string, rType reflect.Type, typeDecl string, creators ...DispatchFunction) ObjectType

// NewNamedType should be used to register an alias for another type.
var NewNamedType func(name, typeDecl string) Type

// NewObjectType creates a new ObjectType with the given name by parsing the provided pcore type definition.
//
// When no creators are given, two constructor functions will be generated based on the attributes given in the
// type definition. The instance created by those functions will always be a struct types.attributeSlice.
//
// Creators:
// A creator is responsible for creating an instance of the new type. The instance must implement the
// PuppetObject interface and its PType() function must return the type that is created by this function.
//
// If one creator is given, then this creator can either be nil to prevent the construction of the default
// creators, or a creator that expects the args passed to be positional arguments for creating an instance.
//
// If two creators are given, then the first creator must be a positional argument creator and the second a
// named argument creator. The named argument creator will always receive exactly one argument of type
// OrderedMap.
//
// A special creator, the types.NoPositionalCreator can be used as the first creator when only a named argument
// creator is desired.
var NewObjectType func(name, typeDecl string, creators ...DispatchFunction) ObjectType

// WrapReflectedType returns the pcore Type for a reflect.Type
var WrapReflectedType func(c Context, rt reflect.Type) (Type, error)

func getPrefix(pfx interface{}) string {
	name := ``
	if s, ok := pfx.(string); ok {
		name = s
	} else if f, ok := pfx.(func() string); ok {
		name = f()
	}
	return name
}
