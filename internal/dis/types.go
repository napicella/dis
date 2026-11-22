package dis

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

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

// Repo is a git repository declared in a distro's repos: map.
type Repo struct {
	// URL is the clone URL, passed to git as is.
	URL string `yaml:"url"`
	// Path is where the repo is cloned. Defaults to ~/<name>; ~ and ${home}
	// are expanded and relative paths are resolved against the home directory.
	Path string `yaml:"path"`
	// Ref is an optional branch or tag to clone. Defaults to the remote's
	// default branch.
	Ref string `yaml:"ref"`
}

// DistroConfig is the structure of a distro YAML file.
// Sources is a plain list of folder paths; the namespace for each installer
// comes from its own provides: field.
type DistroConfig struct {
	OS string `yaml:"os"`
	// Repos lists the git repositories the distro's sources live in, keyed by
	// name. Sources, preconditions and parameters reference them as
	// ${repos.<name>}; "self" is implicit and is the repo holding the distro file.
	Repos    map[string]Repo `yaml:"repos"`
	Sources  []string        `yaml:"sources"`
	Packages []string        `yaml:"packages"`

	// Parameters is the single source of truth for all config values in this
	// distro. Each value can be a plain string (global, available to every
	// package that declares it in requires_env) or an object with a value and
	// an optional packages list (scoped, injected only into the listed packages).
	Parameters map[string]ParameterValue `yaml:"parameters"`

	// Preconditions is a list of checks that run before any installer.
	Preconditions []Precondition `yaml:"preconditions"`

	// Secrets maps an env var name to where its value is kept. An installer
	// that lists the name in requires_env gets the path of a file holding the
	// value, never the value itself.
	Secrets Secrets `yaml:"secrets"`
}

// secretManagerBWS is Bitwarden Secrets Manager, the only secret manager.
const secretManagerBWS = "bws"

// secretManagers are the keys allowed under secrets:.
var secretManagers = []string{secretManagerBWS}

// SecretRef is where a secret's value is kept: a key in a secret manager.
type SecretRef struct {
	Manager string
	Key     string
}

// Secrets maps an env var name to its SecretRef. In the distro file the names
// are under their manager:
//
//	secrets:
//	  bws:
//	    AUTHELIA_SESSION_SECRET: authelia/session_secret
type Secrets map[string]SecretRef

// UnmarshalYAML reads secrets: as managers, each holding NAME: key entries.
// An unknown manager and a name under two managers are errors.
func (s *Secrets) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: secrets: must be a map of secret managers (%s)", node.Line, strings.Join(secretManagers, ", "))
	}
	out := Secrets{}
	seen := map[string]int{} // manager -> line, as YAML refuses a key twice
	for i := 0; i+1 < len(node.Content); i += 2 {
		k, v := node.Content[i], node.Content[i+1]
		if line, ok := seen[k.Value]; ok {
			return fmt.Errorf("line %d: secrets: %q already defined at line %d", k.Line, k.Value, line)
		}
		seen[k.Value] = k.Line
		if !slices.Contains(secretManagers, k.Value) {
			return fmt.Errorf("line %d: secrets: unknown secret manager %q (known: %s): write secrets: {bws: {NAME: key}}",
				k.Line, k.Value, strings.Join(secretManagers, ", "))
		}
		var group map[string]string
		if err := v.Decode(&group); err != nil {
			return fmt.Errorf("secrets: %s: %w", k.Value, err)
		}
		if err := out.add(k.Value, group); err != nil {
			return fmt.Errorf("line %d: %w", k.Line, err)
		}
	}
	*s = out
	return nil
}

// add puts a manager's group of name -> key into s. A name s already holds
// under another manager is an error.
func (s Secrets) add(manager string, group map[string]string) error {
	for name, key := range group {
		if prev, ok := s[name]; ok {
			return fmt.Errorf("secret %s is under both %s and %s: a name must be under one manager", name, prev.Manager, manager)
		}
		s[name] = SecretRef{Manager: manager, Key: key}
	}
	return nil
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

// ResolvedSource is one entry of a distro's sources list.
type ResolvedSource struct {
	// Declared is the entry as written in the distro file, e.g.
	// "${repos.dotfiles}/tools".
	Declared string
	// Path is the absolute directory Declared resolves to.
	Path string
}

// PackageInfo contains read only info about a package.
// It's a subset of a manifest.
type PackageInfo struct {
	// Provides is the fully-qualified name of this installer, e.g. "common/tools".
	Provides string
	// InstallerPath is the absolute path to the installer .sh file.
	InstallerPath string
	// ConfigsDir is the optional configs folder for this package; empty when
	// not declared.
	ConfigsDir string
}
