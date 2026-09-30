package dis

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mitchellh/go-homedir"
)

// InstallContext holds the resolved domain state needed to execute one or more installers.
type InstallContext struct {
	// Cfg is the distro configuration as declared in the YAML file.
	Cfg DistroConfig
	// DistroDir is the directory that contains the distro YAML file.
	DistroDir string
	// Repos are the distro's repos (including the implicit self repo) with
	// their paths resolved.
	Repos []ResolvedRepo

	// parameters is a flat map of all global configuration values available to
	// installers: static values from the distro file and runtime exports from 
	// prior installers (stored as qualified "pkg:VAR" keys).
	parameters map[string]string
	// scopedParameters maps each package name to the extra parameters declared
	// on its packages entry. Scoped values override globals with the same key.
	scopedParameters map[string]map[string]string
	// manifests is the list of all manifests loaded from the sources defined in the distro.
	manifests []Manifest
	// pkgm is the dependency graph of all installer manifests for this distro.
	pkgm *packageManager
}

// NewInstallContext loads the distro configuration from distroFile, resolves
// installer manifests, builds the package dependency graph, writes the
// binding helper script, and initialises parameters from the static distro
// parameters. Preconditions are not run here.
//
// ${home} and ${repos.<name>} are expanded in sources, precondition scripts and
// parameter values.
func NewInstallContext(distroFile string) (*InstallContext, error) {
	var err error
	distroFile, err = homedir.Expand(distroFile)
	if err != nil {
		return nil, fmt.Errorf("expanding distro file path: %w", err)
	}
	distroFile, err = filepath.Abs(distroFile)
	if err != nil {
		return nil, fmt.Errorf("resolving distro file path: %w", err)
	}

	cfg, err := loadDistro(distroFile)
	if err != nil {
		return nil, err
	}
	distroDir := filepath.Dir(distroFile)

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}

	repos, err := ResolveRepos(cfg, distroDir)
	if err != nil {
		return nil, fmt.Errorf("distro %q: %w", distroFile, err)
	}
	vars := newVarExpander(home, repos)

	var resolvedSources []string
	for _, src := range cfg.Sources {
		src, err = vars.expand(src)
		if err != nil {
			return nil, fmt.Errorf("distro %q: sources: %w", distroFile, err)
		}
		if !filepath.IsAbs(src) {
			resolvedSources = append(resolvedSources, filepath.Clean(filepath.Join(distroDir, src)))
		} else {
			resolvedSources = append(resolvedSources, src)
		}
	}
	manifests, err := loadInstallers(resolvedSources, cfg.OS)
	if err != nil {
		return nil, fmt.Errorf("loading sources: %w", err)
	}
	pkgm, err := newPackageManager(manifests)
	if err != nil {
		return nil, fmt.Errorf("building package graph: %w", err)
	}

	// Resolve all script paths in preconditions and config generators to
	// absolute paths so callers never need to handle relative paths.
	for i, pc := range cfg.Preconditions {
		script, err := vars.expand(pc.Script)
		if err != nil {
			return nil, fmt.Errorf("distro %q: preconditions: %w", distroFile, err)
		}
		if !filepath.IsAbs(script) {
			script = filepath.Clean(filepath.Join(distroDir, script))
		}
		cfg.Preconditions[i].Script = script
	}

	// Split parameters into globals and per-package scoped maps.
	params := make(map[string]string, len(cfg.Parameters))
	scopedParameters := make(map[string]map[string]string)
	for k, pv := range cfg.Parameters {
		expanded, err := vars.expand(pv.Value)
		if err != nil {
			return nil, fmt.Errorf("distro %q: parameters.%s: %w", distroFile, k, err)
		}
		if len(pv.Packages) == 0 {
			// Global parameter — available to every package.
			params[k] = expanded
		} else {
			// Scoped parameter — inject only into the listed packages.
			for _, pkgName := range pv.Packages {
				if scopedParameters[pkgName] == nil {
					scopedParameters[pkgName] = make(map[string]string)
				}
				scopedParameters[pkgName][k] = expanded
			}
		}
	}

	return &InstallContext{
		Cfg:              cfg,
		DistroDir:        distroDir,
		Repos:            repos,
		manifests:        manifests,
		pkgm:             pkgm,
		parameters:       params,
		scopedParameters: scopedParameters,
	}, nil
}

// NewInstallContextWithCache is like NewInstallContext but also loads the
// persistent exports cache into ic.parameters. Use this in the install/run
// command so that packages skipped as already-installed still contribute
// their exported values to downstream installers.
func NewInstallContextWithCache(distroFile string) (*InstallContext, error) {
	ic, err := NewInstallContext(distroFile)
	if err != nil {
		return nil, err
	}
	cache, err := ReadExportsCache()
	if err != nil {
		return nil, fmt.Errorf("loading exports cache: %w", err)
	}
	for k, v := range cache {
		// Only populate if not already set by a generator or static param.
		if ic.parameters[k] == "" {
			ic.parameters[k] = v
		}
	}
	return ic, nil
}

// ResolveInstallOrder returns the full ordered list of manifests to install
// for the packages declared in ic.Cfg, respecting transitive dependencies.
func (ic *InstallContext) ResolveInstallOrder() ([]Manifest, error) {
	return ic.pkgm.depsForAll(ic.Cfg.Packages)
}

// ListAvailablePackages returns the list of all the packages that have been loaded from the provided sources.
func (ic *InstallContext) ListAvailablePackages() []PackageInfo {
	var pkgInfos []PackageInfo
	for _, v := range ic.manifests {
		pkgInfos = append(pkgInfos, PackageInfo{
			Provides:      v.Provides,
			InstallerPath: v.InstallerPath,
			ConfigsDir:    v.ConfigsDir,
		})
	}
	return pkgInfos
}

// FindPackage returns the loaded package named name. A name without a "/" may
// also be a short name, the part after the last "/" (e.g. "herdr" for
// "tools/herdr"), as long as a single package has it. A short name shared by
// several packages returns an *AmbiguousPackageError listing them; a name that
// matches nothing returns a *PackageNotFoundError with "Did you mean this?"
// suggestions.
func (ic *InstallContext) FindPackage(name string) (PackageInfo, error) {
	if m, ok := ic.pkgm.get(name); ok {
		return PackageInfo{Provides: m.Provides, InstallerPath: m.InstallerPath, ConfigsDir: m.ConfigsDir}, nil
	}
	if !strings.Contains(name, "/") {
		var matches []PackageInfo
		for _, p := range ic.ListAvailablePackages() {
			if p.Provides[strings.LastIndex(p.Provides, "/")+1:] == name {
				matches = append(matches, p)
			}
		}
		switch len(matches) {
		case 0:
		case 1:
			return matches[0], nil
		default:
			names := make([]string, 0, len(matches))
			for _, p := range matches {
				names = append(names, p.Provides)
			}
			slices.Sort(names)
			return PackageInfo{}, &AmbiguousPackageError{Name: name, Matches: names}
		}
	}
	return PackageInfo{}, ic.packageNotFound(name)
}

// packageNotFound builds the error for a lookup of name that matched nothing,
// suggesting the closest loaded package names.
func (ic *InstallContext) packageNotFound(name string) *PackageNotFoundError {
	var names []string
	for _, p := range ic.ListAvailablePackages() {
		names = append(names, p.Provides)
	}
	return &PackageNotFoundError{Name: name, Suggestions: suggestPackages(name, names)}
}

// envForInstaller returns the env var map for the given installer, resolved
// from rc.parameters and rc.scopedParameters. RequiresEnv entries come in
// four forms:
//
//  1. Bare name ("FOO"): looked up first in the package's scoped parameters,
//     then in rc.parameters (globals).
//  2. Bare glob ("FOO_*"): all keys matching the prefix are collected from
//     scoped parameters first, then globals (scoped wins on conflict).
//  3. Qualified name ("pkg:VAR"): looked up in rc.parameters under the
//     qualified key and injected under the bare name (VAR).
//  4. Qualified glob ("pkg:PREFIX*"): all rc.parameters keys matching the
//     qualified prefix are injected under their bare names.
func (rc *InstallContext) envForInstaller(manifest Manifest) (map[string]string, error) {
	env := make(map[string]string, len(manifest.RequiresEnv))
	for _, envVar := range manifest.RequiresEnv {
		if strings.Contains(envVar, ":") {
			parts := strings.SplitN(envVar, ":", 2)
			varPart := parts[1]

			if strings.HasSuffix(varPart, "*") {
				prefix := envVar[:len(envVar)-1]
				matched := 0
				for k, v := range rc.parameters {
					if strings.HasPrefix(k, prefix) {
						bareKey := strings.SplitN(k, ":", 2)[1]
						env[bareKey] = v
						matched++
					}
				}
				if matched == 0 {
					return nil, fmt.Errorf(
						"installer %q requires %q but no matching exports were found",
						manifest.Provides, envVar,
					)
				}
			} else {
				val := rc.parameters[envVar]
				if val == "" {
					return nil, fmt.Errorf(
						"installer %q requires %q but it has not been exported by %q (is it in depends_on and was it run?)",
						manifest.Provides, envVar, parts[0],
					)
				}
				env[varPart] = val
			}
			continue
		}

		if strings.HasSuffix(envVar, "*") {
			prefix := envVar[:len(envVar)-1]
			matched := 0
			// Globals first, then scoped overrides.
			for k, v := range rc.parameters {
				if strings.HasPrefix(k, prefix) {
					env[k] = v
					matched++
				}
			}
			for k, v := range rc.scopedParameters[manifest.Provides] {
				if strings.HasPrefix(k, prefix) {
					env[k] = v
					matched++
				}
			}
			if matched == 0 {
				return nil, fmt.Errorf(
					"installer %q requires %q but no parameters match that prefix",
					manifest.Provides, envVar,
				)
			}
			continue
		}

		// Bare name: scoped parameters take priority over globals.
		val := rc.scopedParameters[manifest.Provides][envVar]
		if val == "" {
			val = rc.parameters[envVar]
		}
		if val == "" {
			return nil, fmt.Errorf(
				"installer %q requires env var %q but it is not set in distro parameters",
				manifest.Provides, envVar,
			)
		}
		env[envVar] = val
	}
	return env, nil
}

// envForPrecondition returns the env var map for the given precondition,
// injecting only the variables declared in pc.Uses from rc.parameters.
func (rc *InstallContext) envForPrecondition(pc Precondition) (map[string]string, error) {
	env := make(map[string]string, len(pc.Uses))
	for _, varName := range pc.Uses {
		val := rc.parameters[varName]
		if val == "" {
			return nil, fmt.Errorf(
				"precondition script %q requires parameter %q but it is not defined in distro parameters",
				pc.Script, varName,
			)
		}
		env[varName] = val
	}
	return env, nil
}

// addExports reads the exportsFilePath and merges the exported values into
// rc.parameters under qualified keys ("pkg:VAR"), making them available to
// downstream installers via requires_env. It also persists the values to the
// exports cache so they are available in future runs when the package is skipped.
func (rc *InstallContext) addExports(exportsFilePath, providerPkg string) error {
	data, err := os.ReadFile(exportsFilePath)
	if err != nil {
		return err
	}
	newEntries := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 1 {
			return fmt.Errorf("invalid exports line %q (expected KEY=value)", line)
		}
		key := strings.TrimSpace(line[:idx])
		val := line[idx+1:]
		qualifiedKey := providerPkg + ":" + key
		rc.parameters[qualifiedKey] = val
		newEntries[qualifiedKey] = val
	}
	return UpdateExportsCache(newEntries)
}
