// Package dis provides the core domain logic for installing packages defined
// in a distro YAML file.
//
// # Entry points
//
// Start with [NewInstallContext] to load a distro file, resolve installer
// manifests, and build the dependency graph:
//
//	ic, err := dis.NewInstallContext("/path/to/distro.yml")
//
// Then create an [Installer] bound to the target machine:
//
//	runner := dis.NewInstaller(sys.Machine())
//
// Use the installer to run config generators, preconditions, and individual
// installers against the context:
//
//	runner.RunGenerators(ctx, ic)
//	runner.RunPreconditions(ctx, ic)
//	runner.RunInstaller(ctx, ic, "my-package")
//
// To resolve the full ordered install list without executing anything, use:
//
//	manifests, err := ic.ResolveInstallOrder()
package dis
