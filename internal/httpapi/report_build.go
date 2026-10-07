package httpapi

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

var reportBuildOnce sync.Once
var reportBuildID string

// Hash once at handler startup, never on assembly's hot path. The executable
// distinguishes local modified builds even when their VCS revision is equal.
func reportBuildVersion() string {
	reportBuildOnce.Do(func() {
		commit := "unversioned"
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
				}
			}
		}
		path, err := os.Executable()
		if err == nil {
			file, err := os.Open(path)
			if err == nil {
				defer file.Close()
				hash := sha256.New()
				if _, err := io.Copy(hash, file); err == nil {
					reportBuildID = fmt.Sprintf("%s:sha256:%x", commit, hash.Sum(nil))
					return
				}
			}
		}
		// If executable metadata is unavailable, never claim a restarted process
		// is the same unknown build as the one that assembled a saved report.
		reportBuildID = commit + ":process:" + time.Now().UTC().Format(time.RFC3339Nano)
	})
	return reportBuildID
}
