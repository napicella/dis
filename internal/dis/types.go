package dis

import "gopkg.in/yaml.v3"

// PackageEntry is one item in the distro packages list.
// It can be unmarshalled from a plain string:
//
//	- mypkg/foo
//
// or from a map with an optional scoped parameters block:
//
//	- name: mypkg/foo
//	  parameters:
//	    KEY: value
//
// or with multiple names sharing the same parameters:
//
//	- names: [mypkg/foo, mypkg/bar]
//	  parameters:
//	    KEY: value
type PackageEntry struct {
	// Name is a single package name (mutually exclusive with Names).
	Name string `yaml:"name"`
	// Names is a list of package names that share the same scoped parameters.
	Names []string `yaml:"names"`
	// Parameters are injected only into the packages listed in Name/Names.
	// They override globals with the same key.
	Parameters map[string]string `yaml:"parameters"`
}

// ResolvedNames returns the effective list of package names for this entry.
func (e PackageEntry) ResolvedNames() []string {
	if e.Name != "" {
		return []string{e.Name}
	}
	return e.Names
}

// UnmarshalYAML lets a PackageEntry be written as a bare string in YAML,
// preserving full backward-compatibility with plain list entries.
func (e *PackageEntry) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		e.Name = value.Value
		return nil
	}
	// Map form — delegate to the default struct decoder using an alias to
	// avoid infinite recursion.
	type plain PackageEntry
	return value.Decode((*plain)(e))
}

// Precondition is a check script that runs before any installer.
// The script receives only the variables listed in Uses as env vars.
// The script must exit 0 on success, non-zero on failure.
type Precondition struct {
	// Script is the path to the shell script to run.
	Script string `yaml:"script"`
	// Uses lists the parameter names from the distro's Parameters map that
	// should be injected into the script's environment.
	Uses []string `yaml:"uses"`
}

// ConfigGenerator is a shell script that runs before any installer and
// produces dynamic KEY=VALUE pairs (one per line on stdout). Its output is
// merged into the distro Parameters map so that installers can declare the
// generated keys in requires_env and have them validated and injected like
// static parameters.
type ConfigGenerator struct {
	Script string `yaml:"script"`
}

// DistroConfig is the structure of a distro YAML file.
// Sources is a plain list of folder paths; the namespace for each installer
// comes from its own provides: field.
type DistroConfig struct {
	OS       string         `yaml:"os"`
	Sources  []string       `yaml:"sources"`
	Packages []PackageEntry `yaml:"packages"`

	// Parameters is the single source of truth for all global config values
	// in this distro. Each installer declares which parameters it needs via
	// requires_env in its manifest. Preconditions declare their needs via Uses.
	// Only declared parameters are injected.
	// For package-scoped parameters attach a parameters block to the relevant
	// entry in Packages instead.
	Parameters map[string]string `yaml:"parameters"`

	// ConfigGenerators is a list of shell scripts that run before any
	// installer. Each script prints KEY=VALUE lines to stdout; disgo merges
	// the output into Parameters.
	ConfigGenerators []ConfigGenerator `yaml:"config_generators"`

	// Preconditions is a list of checks that run before any installer.
	Preconditions []Precondition `yaml:"preconditions"`
}

// WorkspacePackage describes one package entry within a dis.workspace file.
// After loading, Root and Configs are always absolute paths.
type WorkspacePackage struct {
	// Root is the absolute path to the package root directory.
	Root string `yaml:"root"`
	// Configs is the optional absolute path to the configs directory for this
	// package. Empty when not declared.
	Configs string `yaml:"configs"`
}

// WorkspaceConfig is the structure of a dis.workspace file.
// When present in a source directory it tells dis how to map sub-directories
// to package roots and their associated configs.
type WorkspaceConfig struct {
	Packages []WorkspacePackage `yaml:"packages"`
}

// Manifest describes a single installer script parsed from a .sh file header.
type Manifest struct {
	// Provides is the fully-qualified name of this installer, e.g. "common/tools".
	Provides string
	// InstallerPath is the absolute path to the installer .sh file.
	InstallerPath string
	Distros       []string
	DependsOn     []string
	// RequiresEnv lists the env vars this installer needs injected at runtime.
	// Entries may be bare ("FOO") or qualified ("pkg:VAR").
	RequiresEnv []string
	// ExportsEnv lists the env var names this installer exports for downstream installers.
	ExportsEnv []string
	// PkgRoot is the directory dis treats as the root for this package.
	// When a dis.ws.yml is present it is the entry's declared root directory;
	// otherwise it is the source directory itself.
	PkgRoot string
	// ConfigsDir is the optional configs folder for this package. Set from the
	// dis.workspace entry; empty when not declared.
	ConfigsDir string
}
