package dis

import (
	"slices"
	"testing"
)

func TestSuggestPackages(t *testing.T) {
	candidates := []string{"tools/neovim", "tools/eza", "common/mise", "common/go", "common/docker"}

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "missing prefix", input: "neovim", want: []string{"tools/neovim"}},
		{name: "swapped letters", input: "tools/noevim", want: []string{"tools/neovim"}},
		{name: "typo in prefix", input: "tols/neovim", want: []string{"tools/neovim"}},
		{name: "case insensitive", input: "NEOVIM", want: []string{"tools/neovim"}},
		{name: "missing prefix short name", input: "mise", want: []string{"common/mise"}},
		{name: "missing prefix two letters", input: "go", want: []string{"common/go"}},
		{name: "typo without prefix", input: "dokcer", want: []string{"common/docker"}},
		{name: "no match", input: "xyzzy", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suggestPackages(tt.input, candidates)
			if !slices.Equal(got, tt.want) {
				t.Errorf("suggestPackages(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSuggestPackagesCapsAndOrders(t *testing.T) {
	candidates := []string{"a/fooo", "b/foo", "c/fo", "d/foob", "e/fxo"}
	got := suggestPackages("foo", candidates)
	want := []string{"b/foo", "a/fooo", "c/fo"}
	if !slices.Equal(got, want) {
		t.Errorf("suggestPackages = %v, want %v", got, want)
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"neovim", "neovim", 0},
		{"neovim", "noevim", 2},
		{"kitten", "sitting", 3},
	}
	for _, tt := range tests {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
