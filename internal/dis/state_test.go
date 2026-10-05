package dis

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStateInstalled(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	s := NewState(dir)

	// Nothing is recorded before the first write, and the dir is not needed.
	if got, err := s.ListInstalled(); err != nil || got != nil {
		t.Fatalf("ListInstalled() = %v, %v, want nil, nil", got, err)
	}
	if err := s.RemoveInstalled("common/b"); err != nil {
		t.Fatalf("RemoveInstalled on a missing file: %v", err)
	}

	for _, p := range []string{"common/b", "common/a", "common/b"} {
		if err := s.RecordInstalled(p); err != nil {
			t.Fatalf("RecordInstalled(%q): %v", p, err)
		}
	}
	if got, want := mustList(t, s), []string{"common/a", "common/b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ListInstalled() = %v, want %v (sorted, each once)", got, want)
	}
	if ok, err := s.IsInstalled("common/a"); err != nil || !ok {
		t.Errorf("IsInstalled(common/a) = %v, %v, want true", ok, err)
	}

	if err := s.RemoveInstalled("common/a"); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.IsInstalled("common/a"); ok {
		t.Error("common/a still installed after RemoveInstalled")
	}
	if got, want := mustList(t, s), []string{"common/b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ListInstalled() = %v, want %v", got, want)
	}

	if _, err := os.Stat(filepath.Join(dir, installedFile)); err != nil {
		t.Errorf("state file not in the State dir: %v", err)
	}
}

func TestStateExports(t *testing.T) {
	s := NewState(filepath.Join(t.TempDir(), "state"))

	if got, err := s.ReadExports(); err != nil || len(got) != 0 {
		t.Fatalf("ReadExports() = %v, %v, want empty", got, err)
	}
	if err := s.UpdateExports(map[string]string{"a:X": "1", "b:Y": "2"}); err != nil {
		t.Fatal(err)
	}
	// New values replace old ones for the same key; other keys are kept.
	if err := s.UpdateExports(map[string]string{"a:X": "3"}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadExports()
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"a:X": "3", "b:Y": "2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ReadExports() = %v, want %v", got, want)
	}
}

func mustList(t *testing.T, s *State) []string {
	t.Helper()
	got, err := s.ListInstalled()
	if err != nil {
		t.Fatal(err)
	}
	return got
}
