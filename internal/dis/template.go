package dis

// InstallerTemplate returns the content of a template installer script,
// including a manifest header with placeholder values and a stub body.
// This is the single owner of the manifest format knowledge.
func InstallerTemplate() string {
	return `### -- Manifest
### provides: namespace/name
### depends_on: []
### distro: [ubuntu]
### requires_env: []
### exports_env: []
### -- End

source $DIS_BINDING

# TODO: add installer logic here
`
}
