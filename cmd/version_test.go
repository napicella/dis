package cmd

import (
	"runtime/debug"
	"testing"
)

func TestFormatVersion(t *testing.T) {
	const rev = "aa597471234567890abcdef1234567890abcdef0"
	tests := []struct {
		name     string
		settings []debug.BuildSetting
		want     string
	}{
		{"commit and time", []debug.BuildSetting{
			{Key: "vcs.revision", Value: rev},
			{Key: "vcs.time", Value: "2026-10-02T18:51:00+02:00"},
			{Key: "vcs.modified", Value: "false"},
		}, "aa59747 (2026-10-02 16:51 UTC)"},
		{"uncommitted changes", []debug.BuildSetting{
			{Key: "vcs.revision", Value: rev},
			{Key: "vcs.modified", Value: "true"},
		}, "aa59747+modified"},
		{"no build info", []debug.BuildSetting{{Key: "GOOS", Value: "linux"}}, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatVersion(tt.settings); got != tt.want {
				t.Errorf("formatVersion = %q, want %q", got, tt.want)
			}
		})
	}
}
