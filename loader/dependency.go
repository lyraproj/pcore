package loader

import "github.com/lyraproj/pcore/px"

type dependencyLoader struct {
	basicLoader
	loaders []px.ModuleLoader
	index   map[string]px.ModuleLoader
}

func newDependencyLoader(loaders []px.ModuleLoader) px.Loader {
	index := make(map[string]px.ModuleLoader, len(loaders))
	for _, ml := range loaders {
		index[ml.ModuleName()] = ml
	}
	return &dependencyLoader{
		basicLoader: basicLoader{namedEntries: make(map[string]px.LoaderEntry, 32)},
		loaders:     loaders,
		index:       index}
}

func init() {
	px.NewDependencyLoader = newDependencyLoader
}

func (l *dependencyLoader) LoadEntry(c px.Context, name px.TypedName) px.LoaderEntry {
	entry := l.basicLoader.LoadEntry(c, name)
	if entry == nil {
		entry = l.find(c, name)
		if entry == nil {
			entry = &loaderEntry{nil, nil}
		}
		l.SetEntry(name, entry)
	}
	return entry
}

func (l *dependencyLoader) LoaderFor(moduleName string) px.ModuleLoader {
	return l.index[moduleName]
}

func (l *dependencyLoader) find(c px.Context, name px.TypedName) px.LoaderEntry {
	if name.IsQualified() {
		if ml, ok := l.index[name.Parts()[0]]; ok {
			return ml.LoadEntry(c, name)
		}
		return nil
	}

	for _, ml := range l.loaders {
		e := ml.LoadEntry(c, name)
		if !(e == nil || e.Value() == nil) {
			return e
		}
	}
	return nil
}
