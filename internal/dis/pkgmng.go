package dis

import (
	"fmt"

	"github.com/heimdalr/dag"
)

// packageManager holds a dependency graph of installer manifests and
// resolves ordered install lists.
type packageManager struct {
	d *dag.DAG
}

// newPackageManager creates a packageManager from the given manifests, builds
// the dependency graph, and returns it. Returns an error if any manifest has a
// duplicate provides key or references an unknown dependency.
func newPackageManager(manifests []Manifest) (*packageManager, error) {
	d := dag.NewDAG()
	d.Options(dag.Options{
		VertexHashFunc: func(v interface{}) interface{} {
			return v.(Manifest).Provides
		},
	})

	for _, m := range manifests {
		if err := d.AddVertexByID(m.Provides, m); err != nil {
			return nil, fmt.Errorf("conflict: installer %q is already registered", m.Provides)
		}
	}

	for _, m := range manifests {
		for _, dep := range m.DependsOn {
			if err := d.AddEdge(dep, m.Provides); err != nil {
				if _, getErr := d.GetVertex(dep); getErr != nil {
					return nil, fmt.Errorf("installer %q depends on unknown installer %q", m.Provides, dep)
				}
				return nil, fmt.Errorf("adding edge %q -> %q: %w", dep, m.Provides, err)
			}
		}
	}

	return &packageManager{d: d}, nil
}

// get returns the Manifest for the named installer, or false if not found.
func (t *packageManager) get(name string) (Manifest, bool) {
	v, err := t.d.GetVertex(name)
	if err != nil {
		return Manifest{}, false
	}
	return v.(Manifest), true
}

// depsFor returns all transitive dependencies of the named installer in
// topological order (dependencies first), followed by the installer itself.
func (t *packageManager) depsFor(name string) ([]Manifest, error) {
	return t.depsForAll([]string{name})
}

// depsForAll returns the transitive closure of all packages in pkgs in
// topological order, with each installer appearing exactly once.
func (t *packageManager) depsForAll(pkgs []string) ([]Manifest, error) {
	var ordered []string
	visited := make(map[string]bool)
	for _, pkg := range pkgs {
		if _, err := t.d.GetVertex(pkg); err != nil {
			return nil, fmt.Errorf("installer %q not found", pkg)
		}
		if err := topoSort(t, pkg, visited, &ordered); err != nil {
			return nil, err
		}
	}
	result := make([]Manifest, 0, len(ordered))
	for _, id := range ordered {
		result = append(result, t.manifest(id))
	}
	return result, nil
}

// manifest retrieves the Manifest stored under the given id from the DAG.
// Panics if id is not found — callers must verify existence first.
func (t *packageManager) manifest(id string) Manifest {
	v, err := t.d.GetVertex(id)
	if err != nil {
		panic(fmt.Sprintf("pkgm: manifest %q not found in graph: %v", id, err))
	}
	return v.(Manifest)
}

type visit func(dag.Vertexer)

func (t visit) Visit(v dag.Vertexer) {
	t(v)
}

// topoSort performs a post-order DFS from id, appending each node to ordered
// exactly once (guarded by the shared visited map).
func topoSort(t *packageManager, id string, visited map[string]bool, ordered *[]string) error {
	if visited[id] {
		return nil
	}
	visited[id] = true
	m := t.manifest(id)
	for _, dep := range m.DependsOn {
		if err := topoSort(t, dep, visited, ordered); err != nil {
			return err
		}
	}
	*ordered = append(*ordered, id)
	return nil
}
