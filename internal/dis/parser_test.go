package dis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifest(t *testing.T) {
	tests := []struct {
		name            string
		script          string
		wantProvides    string
		wantDependsOn   []string
		wantDistros     []string
		wantRequiresEnv []string
		wantExportsEnv  []string
	}{
		{
			name: "single-line manifest",
			script: `### -- Manifest
### provides: common/foo
### depends_on: [common/mise,common/os-libs]
### distro: [ubuntu]
### requires_env: [FOO, BAR]
### exports_env: [OUT]
### -- End
source $DIS_BINDING
`,
			wantProvides:    "common/foo",
			wantDependsOn:   []string{"common/mise", "common/os-libs"},
			wantDistros:     []string{"ubuntu"},
			wantRequiresEnv: []string{"FOO", "BAR"},
			wantExportsEnv:  []string{"OUT"},
		},
		{
			name: "multi-line requires_env",
			script: `### -- Manifest
### provides: common/bar
### depends_on: [common/mise]
### distro: [all]
### requires_env: [ALPHA, BETA,
###                GAMMA, DELTA,
###                EPSILON]
### -- End
`,
			wantProvides:    "common/bar",
			wantDependsOn:   []string{"common/mise"},
			wantDistros:     []string{"all"},
			wantRequiresEnv: []string{"ALPHA", "BETA", "GAMMA", "DELTA", "EPSILON"},
		},
		{
			name: "multi-line depends_on",
			script: `### -- Manifest
### provides: common/baz
### depends_on: [common/a,
###              common/b,
###              common/c]
### distro: [ubuntu]
### -- End
`,
			wantProvides:  "common/baz",
			wantDependsOn: []string{"common/a", "common/b", "common/c"},
			wantDistros:   []string{"ubuntu"},
		},
		{
			name: "multi-line requires_env with closing bracket on continuation",
			script: `### -- Manifest
### provides: home-server/containers
### depends_on: [home-server/docker]
### distro: [ubuntu]
### requires_env: [DOCKER_MOUNT_FOLDER, WIREGUARD_KEY_PATH,
###                GID_RENDER, GID_ADM, UID_CONTAINER, GID_CONTAINER,
###                HOME_CONTAINERS_SUBNET, HOST_IP, TZ_CONTAINER, DMN_*]
### exports_env: [GID_DOCKER]
### -- End
`,
			wantProvides:  "home-server/containers",
			wantDependsOn: []string{"home-server/docker"},
			wantDistros:   []string{"ubuntu"},
			wantRequiresEnv: []string{
				"DOCKER_MOUNT_FOLDER", "WIREGUARD_KEY_PATH",
				"GID_RENDER", "GID_ADM", "UID_CONTAINER", "GID_CONTAINER",
				"HOME_CONTAINERS_SUBNET", "HOST_IP", "TZ_CONTAINER", "DMN_*",
			},
			wantExportsEnv: []string{"GID_DOCKER"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "installer.sh")
			if err := os.WriteFile(path, []byte(tt.script), 0o644); err != nil {
				t.Fatalf("writing temp script: %v", err)
			}

			m, ok, err := parseManifest(path, dir, "")
			if err != nil {
				t.Fatalf("parseManifest error: %v", err)
			}
			if !ok {
				t.Fatal("parseManifest returned ok=false; expected a valid manifest")
			}

			if m.Provides != tt.wantProvides {
				t.Errorf("Provides = %q; want %q", m.Provides, tt.wantProvides)
			}
			assertSliceEqual(t, "DependsOn", m.DependsOn, tt.wantDependsOn)
			assertSliceEqual(t, "Distros", m.Distros, tt.wantDistros)
			assertSliceEqual(t, "RequiresEnv", m.RequiresEnv, tt.wantRequiresEnv)
			assertSliceEqual(t, "ExportsEnv", m.ExportsEnv, tt.wantExportsEnv)
		})
	}
}

func assertSliceEqual(t *testing.T, field string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %v (len %d); want %v (len %d)", field, got, len(got), want, len(want))
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q; want %q", field, i, got[i], want[i])
		}
	}
}
