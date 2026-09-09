package version

import (
	"fmt"
	"runtime"
)

const (
	RepoName = "nagomi-email-parser"
	RepoURL  = "https://github.com/xhos/nagomi-email-parser"
)

var (
	Version   = "dev" // overridden at build time
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

func FullVersion() string {
	return fmt.Sprintf("%s (commit: %s, branch: %s, built: %s, go: %s)",
		Version, GitCommit, GitBranch, BuildTime, runtime.Version())
}
