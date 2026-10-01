package dis

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// findPackageContext loads an install context whose only source provides the
// named packages.
func findPackageContext(t *testing.T, names ...string) *InstallContext {
	t.Helper()
	src := t.TempDir()
	for i, name := range names {
		installer := "### -- Manifest\n### provides: " + name + "\n### depends_on: []\n### distro: [all]\n### -- End\necho hi\n"
		path := filepath.Join(src, "pkg"+string(rune('a'+i))+".sh")
		if err := os.WriteFile(path, []byte(installer), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	distro := filepath.Join(t.TempDir(), "distro.yml")
	yml := "os: ubuntu\nsources:\n  - " + src + "\npackages: []\n"
	if err := os.WriteFile(distro, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	ic, err := NewInstallContext(distro)
	if err != nil {
		t.Fatal(err)
	}
	return ic
}

func TestFindPackage(t *testing.T) {
	ic := findPackageContext(t, "tools/herdr", "common/git", "tools/git", "common/tmux")

	tests := []struct {
		name string
		want string
	}{
		{"tools/herdr", "tools/herdr"},
		{"herdr", "tools/herdr"},
		{"tmux", "common/tmux"},
		// An exact full name is never ambiguous.
		{"tools/git", "tools/git"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ic.FindPackage(tt.name)
			if err != nil {
				t.Fatalf("FindPackage(%q): %v", tt.name, err)
			}
			if got.Provides != tt.want {
				t.Errorf("FindPackage(%q) = %q, want %q", tt.name, got.Provides, tt.want)
			}
		})
	}
}

func TestFindPackageAmbiguous(t *testing.T) {
	ic := findPackageContext(t, "tools/git", "common/git")

	_, err := ic.FindPackage("git")
	var amb *AmbiguousPackageError
	if !errors.As(err, &amb) {
		t.Fatalf("FindPackage(\"git\") error = %v, want *AmbiguousPackageError", err)
	}
	if want := []string{"common/git", "tools/git"}; !reflect.DeepEqual(amb.Matches, want) {
		t.Errorf("Matches = %q, want %q", amb.Matches, want)
	}
}

func TestFindPackageNotFound(t *testing.T) {
	ic := findPackageContext(t, "tools/herdr")

	tests := []struct {
		name            string
		wantSuggestions []string
	}{
		// A typo still only gets suggestions, never a silent match.
		{"hedrr", []string{"tools/herdr"}},
		// Short names match exactly, including case.
		{"Herdr", []string{"tools/herdr"}},
		// A name with a "/" is never resolved as a short name.
		{"other/herdr", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ic.FindPackage(tt.name)
			var nf *PackageNotFoundError
			if !errors.As(err, &nf) {
				t.Fatalf("FindPackage(%q) error = %v, want *PackageNotFoundError", tt.name, err)
			}
			if !reflect.DeepEqual(nf.Suggestions, tt.wantSuggestions) {
				t.Errorf("Suggestions = %q, want %q", nf.Suggestions, tt.wantSuggestions)
			}
		})
	}
}

// depsContext loads an install context whose only source provides the given
// packages, each with the listed dependencies.
func depsContext(t *testing.T, pkgs map[string][]string) *InstallContext {
	t.Helper()
	src := t.TempDir()
	i := 0
	for name, deps := range pkgs {
		depList := ""
		for j, d := range deps {
			if j > 0 {
				depList += ", "
			}
			depList += d
		}
		installer := "### -- Manifest\n### provides: " + name + "\n### depends_on: [" + depList + "]\n### distro: [all]\n### -- End\n"
		path := filepath.Join(src, "pkg"+string(rune('a'+i))+".sh")
		if err := os.WriteFile(path, []byte(installer), 0o644); err != nil {
			t.Fatal(err)
		}
		i++
	}
	distro := filepath.Join(t.TempDir(), "distro.yml")
	yml := "os: ubuntu\nsources:\n  - " + src + "\npackages: []\n"
	if err := os.WriteFile(distro, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	ic, err := NewInstallContext(distro)
	if err != nil {
		t.Fatal(err)
	}
	return ic
}

func TestResolveInstallOrderFor(t *testing.T) {
	ic := depsContext(t, map[string][]string{
		"bundle/tools": {"common/one", "common/two"},
		"common/one":   {},
		"common/two":   {"common/base"},
		"common/base":  {},
		"common/other": {},
	})

	// Dependencies come first, each once, and unrelated packages are left out.
	want := []string{"common/one", "common/base", "common/two", "bundle/tools"}
	for _, name := range []string{"bundle/tools", "tools"} {
		t.Run(name, func(t *testing.T) {
			got, err := ic.ResolveInstallOrderFor(name)
			if err != nil {
				t.Fatalf("ResolveInstallOrderFor(%q): %v", name, err)
			}
			var names []string
			for _, m := range got {
				names = append(names, m.Provides)
			}
			if !reflect.DeepEqual(names, want) {
				t.Errorf("ResolveInstallOrderFor(%q) = %v, want %v", name, names, want)
			}
		})
	}

	t.Run("package without dependencies", func(t *testing.T) {
		got, err := ic.ResolveInstallOrderFor("common/other")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Provides != "common/other" {
			t.Errorf("got %v, want only common/other", got)
		}
	})

	t.Run("unknown package", func(t *testing.T) {
		_, err := ic.ResolveInstallOrderFor("common/nope")
		var nf *PackageNotFoundError
		if !errors.As(err, &nf) {
			t.Fatalf("error = %v, want *PackageNotFoundError", err)
		}
	})
}
