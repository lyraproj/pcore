package impl

import (
	"context"
	"fmt"
	"sync"

	"runtime"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/threadlocal"
	"github.com/lyraproj/pcore/types"
)

type (
	pcoreCtx struct {
		context.Context
		loader       eval.Loader
		logger       eval.Logger
		stack        []issue.Location
		implRegistry eval.ImplementationRegistry
		vars         map[string]interface{}
	}

	systemLocation struct{}
)

func (systemLocation) File() string {
	return ``
}

func (systemLocation) Line() int {
	return 0
}

func (systemLocation) Pos() int {
	return 0
}

var resolvableFunctions = make([]eval.ResolvableFunction, 0, 16)
var resolvableFunctionsLock sync.Mutex

func init() {
	eval.Call = func(c eval.Context, name string, args []eval.Value, block eval.Lambda) eval.Value {
		tn := eval.NewTypedName2(`function`, name, c.Loader().NameAuthority())
		if f, ok := eval.Load(c, tn); ok {
			return f.(eval.Function).Call(c, block, args...)
		}
		panic(issue.NewReported(eval.UnknownFunction, issue.SEVERITY_ERROR, issue.H{`name`: tn.String()}, c.StackTop()))
	}

	eval.AddTypes = addTypes

	eval.CurrentContext = func() eval.Context {
		if ctx, ok := threadlocal.Get(eval.PuppetContextKey); ok {
			return ctx.(eval.Context)
		}
		_, file, line, _ := runtime.Caller(1)
		panic(issue.NewReported(eval.NoCurrentContext, issue.SEVERITY_ERROR, issue.NO_ARGS, issue.NewLocation(file, line, 0)))
	}

	eval.RegisterGoFunction = func(function eval.ResolvableFunction) {
		resolvableFunctionsLock.Lock()
		resolvableFunctions = append(resolvableFunctions, function)
		resolvableFunctionsLock.Unlock()
	}

	eval.ResolveResolvables = resolveResolvables
}

func NewContext(loader eval.Loader, logger eval.Logger) eval.Context {
	return WithParent(context.Background(), loader, logger, newImplementationRegistry())
}

func WithParent(parent context.Context, loader eval.Loader, logger eval.Logger, ir eval.ImplementationRegistry) eval.Context {
	var c *pcoreCtx
	ir = newParentedImplementationRegistry(ir)
	if cp, ok := parent.(pcoreCtx); ok {
		c = cp.clone()
		c.Context = parent
		c.loader = loader
		c.logger = logger
	} else {
		c = &pcoreCtx{Context: parent, loader: loader, logger: logger, stack: make([]issue.Location, 0, 8), implRegistry: ir}
	}
	return c
}

func addTypes(c eval.Context, types ...eval.Type) {
	l := c.DefiningLoader()
	rts := make([]eval.ResolvableType, 0, len(types))
	for _, t := range types {
		l.SetEntry(eval.NewTypedName(eval.NsType, t.Name()), eval.NewLoaderEntry(t, nil))
		if rt, ok := t.(eval.ResolvableType); ok {
			rts = append(rts, rt)
		}
	}
	ResolveTypes(c, rts...)
}

func (c *pcoreCtx) DefiningLoader() eval.DefiningLoader {
	l := c.loader
	for {
		if dl, ok := l.(eval.DefiningLoader); ok {
			return dl
		}
		if pl, ok := l.(eval.ParentedLoader); ok {
			l = pl.Parent()
			continue
		}
		panic(`No defining loader found in context`)
	}
}

func (c *pcoreCtx) Delete(key string) {
	if c.vars != nil {
		delete(c.vars, key)
	}
}

func (c *pcoreCtx) DoWithLoader(loader eval.Loader, doer eval.Doer) {
	saveLoader := c.loader
	defer func() {
		c.loader = saveLoader
	}()
	c.loader = loader
	doer()
}

func (c *pcoreCtx) Error(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported {
	if location == nil {
		location = c.StackTop()
	}
	return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
}

func (c *pcoreCtx) Fork() eval.Context {
	s := make([]issue.Location, len(c.stack))
	copy(s, c.stack)
	clone := c.clone()
	clone.loader = eval.NewParentedLoader(clone.loader)
	clone.implRegistry = newParentedImplementationRegistry(clone.implRegistry)
	clone.stack = s

	if c.vars != nil {
		cv := make(map[string]interface{}, len(c.vars))
		for k, v := range c.vars {
			cv[k] = v
		}
		clone.vars = cv
	}
	return clone
}

func (c *pcoreCtx) Fail(message string) issue.Reported {
	return c.Error(nil, eval.Failure, issue.H{`message`: message})
}

func (c *pcoreCtx) Get(key string) (interface{}, bool) {
	if c.vars != nil {
		if v, ok := c.vars[key]; ok {
			return v, true
		}
	}
	return nil, false
}

func (c *pcoreCtx) ImplementationRegistry() eval.ImplementationRegistry {
	return c.implRegistry
}

func (c *pcoreCtx) Loader() eval.Loader {
	return c.loader
}

func (c *pcoreCtx) Logger() eval.Logger {
	return c.logger
}

func (c *pcoreCtx) ParseType(typeString eval.Value) eval.Type {
	if sv, ok := typeString.(eval.StringValue); ok {
		return c.ParseType2(sv.String())
	}
	panic(types.NewIllegalArgumentType(`ParseType`, 0, `String`, typeString))
}

func (c *pcoreCtx) ParseType2(str string) eval.Type {
	t, err := types.Parse(str)
	if err != nil {
		panic(err)
	}
	if pt, ok := t.(eval.ResolvableType); ok {
		return pt.Resolve(c)
	}
	panic(fmt.Errorf(`expression "%s" does no resolve to a Type`, str))
}

func (c *pcoreCtx) Reflector() eval.Reflector {
	return types.NewReflector(c)
}

func resolveResolvables(c eval.Context) {
	l := c.Loader().(eval.DefiningLoader)
	ts := types.PopDeclaredTypes()
	for _, rt := range ts {
		l.SetEntry(eval.NewTypedName(eval.NsType, rt.Name()), eval.NewLoaderEntry(rt, nil))
	}

	for _, mp := range types.PopDeclaredMappings() {
		c.ImplementationRegistry().RegisterType(mp.T, mp.R)
	}

	ResolveTypes(c, ts...)

	ctors := types.PopDeclaredConstructors()
	for _, ct := range ctors {
		rf := eval.BuildFunction(ct.Name, ct.LocalTypes, ct.Creators)
		l.SetEntry(eval.NewTypedName(eval.NsConstructor, rf.Name()), eval.NewLoaderEntry(rf.Resolve(c), nil))
	}

	fs := popDeclaredGoFunctions()
	for _, rf := range fs {
		l.SetEntry(eval.NewTypedName(eval.NsFunction, rf.Name()), eval.NewLoaderEntry(rf.Resolve(c), nil))
	}
}

func (c *pcoreCtx) Set(key string, value interface{}) {
	if c.vars == nil {
		c.vars = map[string]interface{}{key: value}
	} else {
		c.vars[key] = value
	}
}

func (c *pcoreCtx) SetLoader(loader eval.Loader) {
	c.loader = loader
}

func (c *pcoreCtx) Stack() []issue.Location {
	return c.stack
}

func (c *pcoreCtx) StackPop() {
	c.stack = c.stack[:len(c.stack)-1]
}

func (c *pcoreCtx) StackPush(location issue.Location) {
	c.stack = append(c.stack, location)
}

func (c *pcoreCtx) StackTop() issue.Location {
	s := len(c.stack)
	if s == 0 {
		return &systemLocation{}
	}
	return c.stack[s-1]
}

// clone a new context from this context which is an exact copy except for the parent
// of the clone which is set to the original. It is used internally by Fork
func (c *pcoreCtx) clone() *pcoreCtx {
	clone := &pcoreCtx{}
	*clone = *c
	clone.Context = c
	return clone
}

func ResolveTypes(c eval.Context, types ...eval.ResolvableType) {
	l := c.DefiningLoader()
	typeSets := make([]eval.TypeSet, 0)
	allAnnotated := make([]eval.Annotatable, 0, len(types))
	for _, rt := range types {
		t := rt.Resolve(c)
		if ts, ok := t.(eval.TypeSet); ok {
			typeSets = append(typeSets, ts)
		} else {
			var ot eval.ObjectType
			if ot, ok = t.(eval.ObjectType); ok {
				if ctor := ot.Constructor(c); ctor != nil {
					l.SetEntry(eval.NewTypedName(eval.NsConstructor, t.Name()), eval.NewLoaderEntry(ctor, nil))
				}
			}
		}
		if a, ok := t.(eval.Annotatable); ok {
			allAnnotated = append(allAnnotated, a)
		}
	}

	for _, ts := range typeSets {
		allAnnotated = resolveTypeSet(c, l, ts, allAnnotated)
	}

	// Validate type annotations
	for _, a := range allAnnotated {
		a.Annotations(c).EachValue(func(v eval.Value) {
			v.(eval.Annotation).Validate(c, a)
		})
	}
}

func resolveTypeSet(c eval.Context, l eval.DefiningLoader, ts eval.TypeSet, allAnnotated []eval.Annotatable) []eval.Annotatable {
	ts.Types().EachValue(func(tv eval.Value) {
		t := tv.(eval.Type)
		if tsc, ok := t.(eval.TypeSet); ok {
			allAnnotated = resolveTypeSet(c, l, tsc, allAnnotated)
		}
		// Types already known to the loader might have been added to a TypeSet. When that
		// happens, we don't want them added again.
		tn := eval.NewTypedName(eval.NsType, t.Name())
		le := l.LoadEntry(c, tn)
		if le == nil || le.Value() == nil {
			if a, ok := t.(eval.Annotatable); ok {
				allAnnotated = append(allAnnotated, a)
			}
			l.SetEntry(tn, eval.NewLoaderEntry(t, nil))
			if ot, ok := t.(eval.ObjectType); ok {
				if ctor := ot.Constructor(c); ctor != nil {
					l.SetEntry(eval.NewTypedName(eval.NsConstructor, t.Name()), eval.NewLoaderEntry(ctor, nil))
				}
			}
		}
	})
	return allAnnotated
}

func popDeclaredGoFunctions() (fs []eval.ResolvableFunction) {
	resolvableFunctionsLock.Lock()
	fs = resolvableFunctions
	if len(fs) > 0 {
		resolvableFunctions = make([]eval.ResolvableFunction, 0, 16)
	}
	resolvableFunctionsLock.Unlock()
	return
}
