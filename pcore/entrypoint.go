package pcore

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/impl"
	"github.com/lyraproj/pcore/threadlocal"
	"github.com/lyraproj/pcore/types"

	// Ensure that StaticLoader is initialized
	_ "github.com/lyraproj/pcore/loader"
)

type (
	pcoreImpl struct {
		lock              sync.RWMutex
		logger            eval.Logger
		systemLoader      eval.Loader
		environmentLoader eval.Loader
		settings          map[string]*setting
	}
)

var staticLock sync.Mutex
var puppet = &pcoreImpl{settings: make(map[string]*setting, 32)}
var topImplRegistry eval.ImplementationRegistry

func init() {
	eval.Puppet = puppet
	puppet.DefineSetting(`environment`, types.DefaultStringType(), types.WrapString(`production`))
	puppet.DefineSetting(`environmentpath`, types.DefaultStringType(), nil)
	puppet.DefineSetting(`module_path`, types.DefaultStringType(), nil)
	puppet.DefineSetting(`strict`, types.NewEnumType([]string{`off`, `warning`, `error`}, true), types.WrapString(`warning`))
	puppet.DefineSetting(`tasks`, types.DefaultBooleanType(), types.WrapBoolean(false))
	puppet.DefineSetting(`workflow`, types.DefaultBooleanType(), types.WrapBoolean(false))
}

func InitializePuppet() {
	// First call initializes the static loader. There can be only one since it receives
	// most of its contents from Go init() functions
	staticLock.Lock()
	defer staticLock.Unlock()

	if puppet.logger != nil {
		return
	}

	puppet.logger = eval.NewStdLogger()

	eval.RegisterResolvableType(types.NewTypeAliasType(`Pcore::MemberName`, nil, types.TypeMemberName))
	eval.RegisterResolvableType(types.NewTypeAliasType(`Pcore::SimpleTypeName`, nil, types.TypeSimpleTypeName))
	eval.RegisterResolvableType(types.NewTypeAliasType(`Pcore::typeName`, nil, types.TypeTypeName))
	eval.RegisterResolvableType(types.NewTypeAliasType(`Pcore::QRef`, nil, types.TypeQualifiedReference))

	c := impl.NewContext(eval.StaticLoader().(eval.DefiningLoader), puppet.logger)
	eval.ResolveResolvables(c)
	topImplRegistry = c.ImplementationRegistry()
}

func (p *pcoreImpl) Reset() {
	p.lock.Lock()
	p.systemLoader = nil
	p.environmentLoader = nil
	for _, s := range p.settings {
		s.reset()
	}
	p.lock.Unlock()
}

func (p *pcoreImpl) SetLogger(logger eval.Logger) {
	p.logger = logger
}

func (p *pcoreImpl) SystemLoader() eval.Loader {
	p.lock.Lock()
	p.ensureSystemLoader()
	p.lock.Unlock()
	return p.systemLoader
}

func (p *pcoreImpl) configuredStaticLoader() eval.Loader {
	return eval.StaticLoader()
}

// not exported, provides unprotected access to shared object
func (p *pcoreImpl) ensureSystemLoader() eval.Loader {
	if p.systemLoader == nil {
		p.systemLoader = eval.NewParentedLoader(p.configuredStaticLoader())
	}
	return p.systemLoader
}

func (p *pcoreImpl) EnvironmentLoader() eval.Loader {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.environmentLoader == nil {
		p.ensureSystemLoader()
		envLoader := p.systemLoader // TODO: Add proper environment loader
		s := p.settings[`module_path`]
		mds := make([]eval.ModuleLoader, 0)
		lds := []eval.PathType{eval.PuppetFunctionPath, eval.PuppetDataTypePath, eval.PlanPath, eval.TaskPath}
		if s.isSet() {
			modulesPath := s.get().String()
			fis, err := ioutil.ReadDir(modulesPath)
			if err == nil {
				for _, fi := range fis {
					if fi.IsDir() && eval.IsValidModuleName(fi.Name()) {
						ml := eval.NewFileBasedLoader(envLoader, filepath.Join(modulesPath, fi.Name()), fi.Name(), lds...)
						mds = append(mds, ml)
					}
				}
			}
		}
		if len(mds) > 0 {
			p.environmentLoader = eval.NewDependencyLoader(mds)
		} else {
			p.environmentLoader = envLoader
		}
	}
	return p.environmentLoader
}

func (p *pcoreImpl) Loader(key string) eval.Loader {
	envLoader := p.EnvironmentLoader()
	if key == `` {
		return envLoader
	}
	if dp, ok := envLoader.(eval.DependencyLoader); ok {
		return dp.LoaderFor(key)
	}
	return nil
}

func (p *pcoreImpl) DefineSetting(key string, valueType eval.Type, dflt eval.Value) {
	s := &setting{name: key, valueType: valueType, defaultValue: dflt}
	if dflt != nil {
		s.set(dflt)
	}
	p.lock.Lock()
	p.settings[key] = s
	p.lock.Unlock()
}

func (p *pcoreImpl) Get(key string, defaultProducer eval.Producer) eval.Value {
	p.lock.RLock()
	v, ok := p.settings[key]
	p.lock.RUnlock()

	if ok {
		if v.isSet() {
			return v.get()
		}
		if defaultProducer == nil {
			return eval.Undef
		}
		return defaultProducer()
	}
	panic(fmt.Sprintf(`Attempt to access unknown setting '%s'`, key))
}

func (p *pcoreImpl) Logger() eval.Logger {
	return p.logger
}

func (p *pcoreImpl) RootContext() eval.Context {
	InitializePuppet()
	c := impl.WithParent(context.Background(), eval.NewParentedLoader(p.EnvironmentLoader()), p.logger, topImplRegistry)
	threadlocal.Init()
	threadlocal.Set(eval.PuppetContextKey, c)
	return c
}

func (p *pcoreImpl) Do(actor func(eval.Context)) {
	p.DoWithParent(p.RootContext(), actor)
}

func (p *pcoreImpl) DoWithParent(parentCtx context.Context, actor func(eval.Context)) {
	InitializePuppet()
	var ctx eval.Context
	if ec, ok := parentCtx.(eval.Context); ok {
		ctx = ec.Fork()
	} else {
		ctx = impl.WithParent(parentCtx, eval.NewParentedLoader(p.EnvironmentLoader()), p.logger, topImplRegistry)
	}
	eval.DoWithContext(ctx, actor)
}

func (p *pcoreImpl) Try(actor func(eval.Context) error) (err error) {
	return p.TryWithParent(p.RootContext(), actor)
}

func (p *pcoreImpl) TryWithParent(parentCtx context.Context, actor func(eval.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if ri, ok := r.(error); ok {
				err = ri
			} else {
				panic(r)
			}
		}
	}()
	p.DoWithParent(parentCtx, func(c eval.Context) {
		err = actor(c)
	})
	return
}

func (p *pcoreImpl) Set(key string, value eval.Value) {
	p.lock.RLock()
	v, ok := p.settings[key]
	p.lock.RUnlock()

	if ok {
		v.set(value)
		return
	}
	panic(fmt.Sprintf(`Attempt to assign unknown setting '%s'`, key))
}
