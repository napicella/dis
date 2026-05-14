package dis

import "gopkg.in/yaml.v3"

// ParameterValue represents a single entry in the distro parameters map.
// It can be unmarshalled from a plain string (global parameter):
//
//	MY_PARAM: my-value
//
// or from a map with an optional packages scope (scoped parameter):
//
//	MY_PARAM:
//	  value: my-value
//	  packages: [mypkg/foo, mypkg/bar]
//
// When Packages is empty the parameter is global — available to every package
// that declares it in requires_env. When Packages is non-empty the parameter
// is injected only into the listed packages.
type ParameterValue struct {
	// Value is the parameter value string.
	Value string
	// Packages is the optional list of package names this parameter is scoped to.
	// Empty means the parameter is global.
	Packages []string
}

// UnmarshalYAML allows ParameterValue to be written as either a bare string
// or as a map with value/packages keys.
func (p *ParameterValue) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		p.Value = value.Value
		return nil
	}
	// Map form — decode into a temporary struct to avoid infinite recursion.
	type plain struct {
		Value    string   `yaml:"value"`
		Packages []string `yaml:"packages"`
	}
	var tmp plain
	if err := value.Decode(&tmp); err != nil {
		return err
	}
	p.Value = tmp.Value
	p.Packages = tmp.Packages
	return nil
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
	OS       string   `yaml:"os"`
	Sources  []string `yaml:"sources"`
	Packages []string `yaml:"packages"`

	// Parameters is the single source of truth for all config values in this
	// distro. Each value can be a plain string (global, available to every
	// package that declares it in requires_env) or an object with a value and
	// an optional packages list (scoped, injected only into the listed packages).
	Parameters map[string]ParameterValue `yaml:"parameters"`

	// ConfigGenerators is a list of shell scripts that run before any
	// installer. Each script prints KEY=VALUE lines to stdout; dis merges
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
