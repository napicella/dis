package dis

import (
	"slices"
	"strings"
)

// maxSuggestions caps the number of package names proposed on a lookup miss.
const maxSuggestions = 3

// suggestPackages returns up to maxSuggestions candidates that name likely
// refers to, best match first. A candidate matches when its basename (the part
// after the last "/") equals name, e.g. "herdr" for "tools/herdr", or when it
// is within a small edit distance of name. Comparisons are case-insensitive.
func suggestPackages(name string, candidates []string) []string {
	type match struct {
		name     string
		rank     int // 0: basename match, 1: edit distance match
		distance int
	}

	lname := strings.ToLower(name)
	// Tolerate fewer typos in short names: 2 edits on "go" or "mise" would
	// rewrite half the word and match unrelated packages (e.g. "jq", "wine").
	maxDistance := 2
	if len([]rune(lname)) <= 4 {
		maxDistance = 1
	}

	var matches []match
	for _, c := range candidates {
		lc := strings.ToLower(c)
		base := lc[strings.LastIndex(lc, "/")+1:]

		// Exact name but missing prefix (e.g. "herdr" for "tools/herdr"): the
		// strongest signal, so it ranks above any typo match.
		if base == lname {
			matches = append(matches, match{name: c, rank: 0})
			continue
		}

		d := levenshtein(lname, lc)
		// Without a "/", the user likely omitted the prefix too (e.g. "hedrr"),
		// so also compare against the basename alone.
		if !strings.Contains(lname, "/") {
			d = min(d, levenshtein(lname, base))
		}
		if d <= maxDistance {
			matches = append(matches, match{name: c, rank: 1, distance: d})
		}
	}

	slices.SortFunc(matches, func(a, b match) int {
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		if a.distance != b.distance {
			return a.distance - b.distance
		}
		return strings.Compare(a.name, b.name)
	})

	var out []string
	for _, m := range matches {
		if !slices.Contains(out, m.name) {
			out = append(out, m.name)
		}
		if len(out) == maxSuggestions {
			break
		}
	}
	return out
}

// levenshtein returns the edit distance between a and b.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}
