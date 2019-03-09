package loader

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

type (
	ContentProvidingLoader interface {
		px.Loader

		GetContent(c px.Context, path string) []byte
	}

	fileBasedLoader struct {
		parentedLoader
		path            string
		moduleName      string
		initPlanName    px.TypedName
		initTaskName    px.TypedName
		initTypeSetName px.TypedName
		paths           map[px.Namespace][]SmartPath
		index           map[string][]string
	}

	SmartPathFactory func(loader px.ModuleLoader, moduleNameRelative bool) SmartPath
)

var SmartPathFactories map[px.PathType]SmartPathFactory = map[px.PathType]SmartPathFactory{
	px.PuppetDataTypePath: newPuppetTypePath,
}

func init() {
	px.NewFileBasedLoader = newFileBasedLoader
}

func newFileBasedLoader(parent px.Loader, path, moduleName string, lds ...px.PathType) px.ModuleLoader {
	paths := make(map[px.Namespace][]SmartPath, len(lds))
	loader := &fileBasedLoader{
		parentedLoader: parentedLoader{
			basicLoader: basicLoader{namedEntries: make(map[string]px.LoaderEntry, 64)},
			parent:      parent},
		path:            path,
		initPlanName:    px.NewTypedName2(px.NsPlan, `init`, parent.NameAuthority()),
		initTaskName:    px.NewTypedName2(px.NsTask, `init`, parent.NameAuthority()),
		initTypeSetName: px.NewTypedName2(px.NsType, `init_typeset`, parent.NameAuthority()),
		moduleName:      moduleName,
		paths:           paths}

	for _, p := range lds {
		path := loader.newSmartPath(p, !(moduleName == `` || moduleName == `environment`))
		if sa, ok := paths[path.Namespace()]; ok {
			paths[path.Namespace()] = append(sa, path)
		} else {
			paths[path.Namespace()] = []SmartPath{path}
		}
	}
	return loader
}

func (l *fileBasedLoader) newSmartPath(pathType px.PathType, moduleNameRelative bool) SmartPath {
	if f, ok := SmartPathFactories[pathType]; ok {
		return f(l, moduleNameRelative)
	}
	panic(px.Error(px.IllegalArgument, issue.H{`function`: `newSmartPath`, `index`: 1, `arg`: pathType}))
}

func newPuppetTypePath(loader px.ModuleLoader, moduleNameRelative bool) SmartPath {
	return NewSmartPath(`types`, `.pp`, loader, px.NsType, moduleNameRelative, false, InstantiatePuppetType)
}

func (l *fileBasedLoader) LoadEntry(c px.Context, name px.TypedName) px.LoaderEntry {
	entry := l.parentedLoader.LoadEntry(c, name)
	if entry == nil {
		entry = l.find(c, name)
		if entry == nil {
			entry = &loaderEntry{nil, nil}
			l.SetEntry(name, entry)
		}
	}
	return entry
}

func (l *fileBasedLoader) ModuleName() string {
	return l.moduleName
}

func (l *fileBasedLoader) Path() string {
	return l.path
}

func (l *fileBasedLoader) isGlobal() bool {
	return l.moduleName == `` || l.moduleName == `environment`
}

func (l *fileBasedLoader) find(c px.Context, name px.TypedName) px.LoaderEntry {
	if name.IsQualified() {
		// The name is in a name space.
		if l.moduleName != `` && l.moduleName != name.Parts()[0] {
			// Then entity cannot possible be in this module unless the name starts with the module name.
			// Note: If "module" represents a "global component", the module_name is empty and cannot match which is
			// ok since such a "module" cannot have namespaced content).
			return nil
		}
		if name.Namespace() == px.NsTask && len(name.Parts()) > 2 {
			// Subdirectories beneath the tasks directory are currently not recognized
			return nil
		}
	} else {
		// The name is in the global name space.
		switch name.Namespace() {
		case px.NsFunction:
			// Can be defined in module using a global name. No action required
		case px.NsPlan:
			if !l.isGlobal() {
				// Global name must be the name of the module
				if l.moduleName != name.Parts()[0] {
					// Global name must be the name of the module
					return nil
				}

				// Look for special 'init' plan
				origins, smartPath := l.findExistingPath(l.initPlanName)
				if smartPath == nil {
					return nil
				}
				return l.instantiate(c, smartPath, name, origins)
			}
		case px.NsTask:
			if !l.isGlobal() {
				// Global name must be the name of the module
				if l.moduleName != name.Parts()[0] {
					// Global name must be the name of the module
					return nil
				}

				// Look for special 'init' task
				origins, smartPath := l.findExistingPath(l.initTaskName)
				if smartPath == nil {
					return nil
				}
				return l.instantiate(c, smartPath, name, origins)
			}
		case px.NsType:
			if !l.isGlobal() {
				// Global name must be the name of the module
				if l.moduleName != name.Parts()[0] {
					// Global name must be the name of the module
					return nil
				}

				// Look for special 'init_typeset' TypeSet
				origins, smartPath := l.findExistingPath(l.initTypeSetName)
				if smartPath == nil {
					return nil
				}
				smartPath.Instantiator()(c, l, name, origins)
				entry := l.GetEntry(name)
				if entry != nil {
					if _, ok := entry.Value().(px.TypeSet); ok {
						return entry
					}
				}
				panic(px.Error(px.NotExpectedTypeset, issue.H{`source`: origins[0], `name`: utils.CapitalizeSegment(l.moduleName)}))
			}
		default:
			return nil
		}
	}

	origins, smartPath := l.findExistingPath(name)
	if smartPath != nil {
		return l.instantiate(c, smartPath, name, origins)
	}

	if !(name.Namespace() == px.NsType && name.IsQualified()) {
		return nil
	}

	// Search for TypeSet using parent name
	tsName := name.Parent()
	for tsName != nil {
		tse := l.GetEntry(tsName)
		if tse == nil {
			tse = l.find(c, tsName)
		}
		if tse != nil && tse.Value() != nil {
			if ts, ok := tse.Value().(px.TypeSet); ok {
				c.DoWithLoader(l, func() {
					ts.(px.ResolvableType).Resolve(c)
				})
				te := l.GetEntry(name)
				if te != nil {
					return te
				}
			}
		}
		tsName = tsName.Parent()
	}
	return nil
}

func (l *fileBasedLoader) findExistingPath(name px.TypedName) (origins []string, smartPath SmartPath) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if paths, ok := l.paths[name.Namespace()]; ok {
		for _, sm := range paths {
			l.ensureIndexed(sm)
			if paths, ok := l.index[name.MapKey()]; ok {
				return paths, sm
			}
		}
	}
	return nil, nil
}

func (l *fileBasedLoader) ensureAllIndexed() {
	l.lock.Lock()
	defer l.lock.Unlock()

	for _, paths := range l.paths {
		for _, sm := range paths {
			l.ensureIndexed(sm)
		}
	}
}

func (l *fileBasedLoader) ensureIndexed(sp SmartPath) {
	if !sp.Indexed() {
		sp.SetIndexed()
		l.addToIndex(sp)
	}
}

func (l *fileBasedLoader) instantiate(c px.Context, smartPath SmartPath, name px.TypedName, origins []string) px.LoaderEntry {
	smartPath.Instantiator()(c, l, name, origins)
	return l.GetEntry(name)
}

func (l *fileBasedLoader) Discover(c px.Context, predicate func(px.TypedName) bool) []px.TypedName {
	l.ensureAllIndexed()
	found := l.parent.Discover(c, predicate)
	added := false
	for k := range l.index {
		tn := px.TypedNameFromMapKey(k)
		if !l.parent.HasEntry(tn) {
			if predicate(tn) {
				found = append(found, tn)
				added = true
			}
		}
	}
	if added {
		sort.Slice(found, func(i, j int) bool { return found[i].MapKey() < found[j].MapKey() })
	}
	return found
}

func (l *fileBasedLoader) GetContent(c px.Context, path string) []byte {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		panic(px.Error(px.UnableToReadFile, issue.H{`path`: path, `detail`: err.Error()}))
	}
	return content
}

func (l *fileBasedLoader) HasEntry(name px.TypedName) bool {
	if l.parent.HasEntry(name) {
		return true
	}

	if paths, ok := l.paths[name.Namespace()]; ok {
		for _, sm := range paths {
			l.ensureIndexed(sm)
			if _, ok := l.index[name.MapKey()]; ok {
				return true
			}
		}
	}
	return false
}

func (l *fileBasedLoader) addToIndex(smartPath SmartPath) {
	if l.index == nil {
		l.index = make(map[string][]string, 64)
	}
	ext := smartPath.Extension()
	noExtension := ext == ``

	generic := smartPath.GenericPath()
	err := filepath.Walk(generic, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if strings.Contains(err.Error(), `no such file or directory`) {
				// A missing path is OK
				err = nil
			}
			return err
		}
		if !info.IsDir() {
			if noExtension || strings.HasSuffix(path, ext) {
				rel, err := filepath.Rel(generic, path)
				if err == nil {
					tn := smartPath.TypedName(l.NameAuthority(), rel)
					if tn != nil {
						if paths, ok := l.index[tn.MapKey()]; ok {
							l.index[tn.MapKey()] = append(paths, path)
						} else {
							l.index[tn.MapKey()] = []string{path}
						}
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		panic(px.Error(px.Failure, issue.H{`message`: err.Error()}))
	}
}
