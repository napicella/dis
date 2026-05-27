package dis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lesiw.io/command"
)

// Installer executes installers, config generators, and precondition scripts
// against a specific machine. It owns the wrapper.sh helper file for the
// lifetime of the session; call Close when done.
type Installer struct {
	// machine is the target on which scripts are executed.
	machine     command.Machine
	wrapperPath string
	// disBinDir is the directory containing the dis binary, added to PATH so
	// that installer scripts can call `dis tools ...` directly.
	disBinDir string
	// Reinstall skips the already-installed check so every package is
	// re-executed regardless of its recorded state.
	Reinstall bool
}

// NewInstaller writes the wrapper.sh helper to a temp file and returns an
// Installer that runs scripts on m. Call Close to remove the temp file.
func NewInstaller(m command.Machine) (*Installer, error) {
	wrapperPath, err := writeWrapper()
	if err != nil {
		return nil, fmt.Errorf("could not write wrapper.sh: %w", err)
	}

	disBinDir := ""
	exe, err := os.Executable()
	if err == nil {
		disBinDir = filepath.Dir(exe)
	}

	return &Installer{machine: m, wrapperPath: wrapperPath, disBinDir: disBinDir}, nil
}

// Close removes the wrapper.sh temp file written at construction time.
func (r *Installer) Close() error {
	return os.Remove(r.wrapperPath)
}

// pathEnv returns a PATH value that prepends disBinDir to the current PATH so
// that installer scripts can call `dis tools ...` directly.
func (r *Installer) pathEnv() string {
	current := os.Getenv("PATH")
	if r.disBinDir == "" {
		return current
	}
	if current == "" {
		return r.disBinDir
	}
	return r.disBinDir + ":" + current
}

// RunGenerators executes all config_generator scripts declared in ic.Cfg,
// parses their stdout as KEY=VALUE lines, and merges the results into
// ic.parameters.
func (r *Installer) RunGenerators(ctx context.Context, ic *InstallContext) error {
	for _, gen := range ic.Cfg.ConfigGenerators {
		fmt.Printf("==> Running config generator: %s\n", gen.Script)

		out, err := command.Read(ctx, r.machine, "/bin/bash", gen.Script)
		if err != nil {
			return fmt.Errorf("config generator %q failed: %w", gen.Script, err)
		}

		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			idx := strings.IndexByte(line, '=')
			if idx < 1 {
				return fmt.Errorf("config generator %q: invalid output line %q (expected KEY=VALUE)", gen.Script, line)
			}
			ic.parameters[strings.TrimSpace(line[:idx])] = line[idx+1:]
		}
	}
	return nil
}

// RunPreconditions executes each precondition script declared in ic.Cfg,
// injecting the variables it declares from ic.parameters.
func (r *Installer) RunPreconditions(ctx context.Context, ic *InstallContext) error {
	for _, pc := range ic.Cfg.Preconditions {
		pcEnvVars, err := ic.envForPrecondition(pc)
		if err != nil {
			return err
		}
		pcEnvVars["DIS_DISTRO"] = ic.Cfg.OS
		pcEnvVars["PATH"] = r.pathEnv()

		fmt.Printf("==> Checking precondition: %s\n", pc.Script)
		pcCtx := command.WithEnv(ctx, pcEnvVars)

		if err := command.Exec(pcCtx, r.machine, r.wrapperPath, pc.Script); err != nil {
			return fmt.Errorf("precondition %q failed: %w", pc.Script, err)
		}
	}
	return nil
}

// RunInstaller looks up pkgName in ic.packages, then executes its installer
// script via the wrapper so that previously-written RC paths are available.
// It is a no-op if the package is already recorded in the state file.
// After the installer finishes, any values it exported via DIS_EXPORTS_FILE are
// merged into ic.parameters under qualified keys ("pkg:VAR") for downstream
// installers. On success the package is recorded in the state file.
func (r *Installer) RunInstaller(ctx context.Context, ic *InstallContext, pkgName string) error {
	if r.Reinstall {
		if err := RemoveInstalled(pkgName); err != nil {
			return fmt.Errorf("removing install state for %q: %w", pkgName, err)
		}
	} else {
		alreadyInstalled, err := IsInstalled(pkgName)
		if err != nil {
			return fmt.Errorf("checking install state for %q: %w", pkgName, err)
		}
		if alreadyInstalled {
			fmt.Printf("==> Skipping %s (already installed)\n", pkgName)
			return nil
		}
	}

	manifest, ok := ic.pkgm.get(pkgName)
	if !ok {
		return fmt.Errorf("package %q not found in any of the configured sources", pkgName)
	}

	installerPath := manifest.InstallerPath
	fmt.Printf("==> Installing %s (%s)\n", manifest.Provides, installerPath)

	exportsFile, err := os.CreateTemp("", "dis-exports-*")
	if err != nil {
		return fmt.Errorf("creating exports file: %w", err)
	}
	exportsFile.Close()
	defer os.Remove(exportsFile.Name())

	envVars := map[string]string{
		"DIS_PKG_ROOT":     manifest.PkgRoot,
		"DIS_INSTALLER":    installerPath,
		"DIS_DISTRO":       ic.Cfg.OS,
		"DIS_EXPORTS_FILE": exportsFile.Name(),
		"PATH":             r.pathEnv(),
	}
	if manifest.ConfigsDir != "" {
		envVars["DIS_CONFIG_FOLDER"] = manifest.ConfigsDir
	}
	pkgEnv, err := ic.envForInstaller(manifest)
	if err != nil {
		return err
	}
	for k, v := range pkgEnv {
		envVars[k] = v
	}

	installerCtx := command.WithEnv(ctx, envVars)
	if err := command.Exec(installerCtx, r.machine, r.wrapperPath, installerPath); err != nil {
		return err
	}

	if len(manifest.ExportsEnv) > 0 {
		if err := ic.addExports(exportsFile.Name(), manifest.Provides); err != nil {
			return fmt.Errorf("reading exports from %q: %w", manifest.Provides, err)
		}
	}

	if err := RecordInstalled(manifest.Provides); err != nil {
		return fmt.Errorf("recording install state for %q: %w", manifest.Provides, err)
	}

	return nil
}
