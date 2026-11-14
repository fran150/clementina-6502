// Package buildinfo reports which build of an emulator executable is running,
// so a local dev build is never mistaken for a release.
package buildinfo

import (
	"runtime/debug"
	"strings"
)

// Version is the release tag, set at link time with
// -ldflags "-X github.com/fran150/clementina-6502/internal/buildinfo.Version=v1.2.3".
// Builds that do not set it are dev builds.
var Version = "dev"

// String returns one line such as "clementina-automation v0.4.0-alpha.1 (commit abc1234)"
// or "clementina-automation dev (commit abc1234 with local changes)". The commit
// comes from the version control information go build embeds.
func String(name string) string {
	line := name + " " + Version
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return line
	}
	var revision string
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision == "" {
		return line
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	details := []string{"commit " + revision}
	if modified {
		details = append(details, "with local changes")
	}
	return line + " (" + strings.Join(details, " ") + ")"
}
