package dis

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
	// distro. Each installer declares which parameters it needs via
	// requires_env in its manifest. Preconditions declare their needs via Uses.
	// Only declared parameters are injected.
	Parameters map[string]string `yaml:"parameters"`

	// ConfigGenerators is a list of shell scripts that run before any
	// installer. Each script prints KEY=VALUE lines to stdout; disgo merges
	// the output into Parameters.
	ConfigGenerators []ConfigGenerator `yaml:"config_generators"`

	// Preconditions is a list of checks that run before any installer.
	Preconditions []Precondition `yaml:"preconditions"`
}

// Manifest describes a single installer script parsed from a .sh file header.
type Manifest struct {
	// Provides is the fully-qualified name of this installer, e.g. "common/tools".
	Provides         string
	RelativeFilepath string
	Distros          []string
	DependsOn        []string
	// RequiresEnv lists the env vars this installer needs injected at runtime.
	// Entries may be bare ("FOO") or qualified ("pkg:VAR").
	RequiresEnv []string
	// ExportsEnv lists the env var names this installer exports for downstream installers.
	ExportsEnv []string
	// SourceDir is the root directory from which RelativeFilepath is relative.
	SourceDir string
}
