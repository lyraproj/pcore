package eval

import (
	"context"
	"github.com/lyraproj/pcore/threadlocal"
	"runtime"

	"github.com/lyraproj/issue/issue"
)

const PuppetContextKey = `puppet.context`

// An Context holds all state during evaluation. Since it contains the stack, each
// thread of execution must use a context of its own. It's expected that multiple
// contexts share common parents for scope and loaders.
//
type Context interface {
	context.Context

	// Delete deletes the given key from the context variable map
	Delete(key string)

	// DefiningLoader returns a Loader that can receive new definitions
	DefiningLoader() DefiningLoader

	// DoWithLoader assigns the given loader to the receiver and calls the doer. The original loader is
	// restored before this call returns.
	DoWithLoader(loader Loader, doer Doer)

	// Error creates a Reported with the given issue code, location, and arguments
	// Typical use is to panic with the returned value
	Error(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported

	// Fail creates a Reported with the EVAL_FAILURE issue code, location from stack top,
	// and the given message
	// Typical use is to panic with the returned value
	Fail(message string) issue.Reported

	// Fork a new context from this context. The fork will have the same scope,
	// loaders, and logger as this context. The stack and the map of context variables will
	// be shallow copied
	Fork() Context

	// Get returns the context variable with the given key together with a bool to indicate
	// if the key was found
	Get(key string) (interface{}, bool)

	// ImplementationRegistry returns the registry that holds mappings between Type and reflect.Type
	ImplementationRegistry() ImplementationRegistry

	// Loader returns the loader of the receiver.
	Loader() Loader

	// Logger returns the logger of the receiver. This will be the same logger as the
	// logger of the evaluator.
	Logger() Logger

	// ParseType parses and evaluates the given Value into a Type. It will panic with
	// an issue.Reported unless the parsing was successful and the result is evaluates
	// to a Type
	ParseType(str Value) Type

	// ParseType2 parses and evaluates the given string into a Type. It will panic with
	// an issue.Reported unless the parsing was successful and the result is evaluates
	// to a Type
	ParseType2(typeString string) Type

	// Reflector returns a Reflector capable of converting to and from reflected values
	// and types
	Reflector() Reflector

	// Set adds or replaces the context variable for the given key with the given value
	Set(key string, value interface{})

	// Permanently change the loader of this context
	SetLoader(loader Loader)

	// Stack returns the full stack. The returned value must not be modified.
	Stack() []issue.Location

	// StackPop pops the last pushed location from the stack
	StackPop()

	// StackPush pushes a location onto the stack. The location is typically the
	// currently evaluated expression.
	StackPush(location issue.Location)

	// StackTop returns the top of the stack
	StackTop() issue.Location
}

// AddTypes Makes the given types known to the loader appointed by this context
var AddTypes func(c Context, types ...Type)

// Call calls a function known to the loader with arguments and an optional
// block.
var Call func(c Context, name string, args []Value, block Lambda) Value

// Resolve types, constructions, or functions that has been recently added
var ResolveResolvables func(c Context)

func DoWithContext(ctx Context, actor func(Context)) {
	if saveCtx, ok := threadlocal.Get(PuppetContextKey); ok {
		defer func() {
			threadlocal.Set(PuppetContextKey, saveCtx)
		}()
	} else {
		threadlocal.Init()
	}
	threadlocal.Set(PuppetContextKey, ctx)
	actor(ctx)
}

var CurrentContext func() Context

func StackTop() issue.Location {
	if ctx, ok := threadlocal.Get(PuppetContextKey); ok {
		return ctx.(Context).StackTop()
	}
	_, file, line, _ := runtime.Caller(2)
	return issue.NewLocation(file, line, 0)
}
