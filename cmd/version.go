package cmd

import (
	"runtime/debug"
	"time"
)

// buildVersion describes the running binary from the build info Go embeds:
// the commit it was built from and that commit's time, with "+modified" for a
// build with uncommitted changes. Without build info (go run,
// -buildvcs=false) it returns "unknown".
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	return formatVersion(info.Settings)
}

func formatVersion(settings []debug.BuildSetting) string {
	var rev, at string
	modified := false
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.time":
			at = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if rev == "" {
		return "unknown"
	}
	v := rev[:min(7, len(rev))]
	if modified {
		v += "+modified"
	}
	if t, err := time.Parse(time.RFC3339, at); err == nil {
		v += " (" + t.UTC().Format("2006-01-02 15:04 UTC") + ")"
	}
	return v
}
