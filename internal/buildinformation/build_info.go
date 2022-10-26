package buildinformation

import (
	"runtime/debug"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

const (
	goarch      string = "GOARCH"
	goos        string = "GOOS"
	vcstype     string = "vcs"
	vcsrevision string = "vcs.revision"
	vcstime     string = "vcs.time"
)

type BuildInfo struct {
	commitSHA       string
	architecture    string
	vcsTime         string
	vcsType         string
	operatingSystem string
	goVersion       string
}

// GetBuildInfo returns build related information of the binary or an error.
func GetBuildInfo() (*BuildInfo, error) {
	b := &BuildInfo{}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, errors.GeneralError("unable to get build info")
	}
	b.goVersion = info.GoVersion
	for _, i := range info.Settings {
		switch i.Key {
		case vcstime:
			b.vcsTime = i.Value
		case goarch:
			b.architecture = i.Value
		case vcstype:
			b.vcsType = i.Value
		case goos:
			b.operatingSystem = i.Value
		case vcsrevision:
			b.commitSHA = i.Value
		}
	}
	return b, nil
}

// GetCommitSha returns the current commitSha (version).
// The returned string can be empty if binary was not built using VCS.
func (b *BuildInfo) GetCommitSHA() string {
	return b.commitSHA
}

// GetArchitecture returns the current architecture.
func (b *BuildInfo) GetArchitecture() string {
	return b.architecture
}

// GetVCSTime returns the current commit time.
// The returned string can be empty if binary was not built using VCS.
func (b *BuildInfo) GetVCSTime() string {
	return b.vcsTime
}

// GetVCSType returns the Version Control System type.
// The returned string can be empty if binary was not built using VCS.
func (b *BuildInfo) GetVCSType() string {
	return b.vcsType
}

// GetOperatingSystem returns the current Operating System.
func (b *BuildInfo) GetOperatingSystem() string {
	return b.operatingSystem
}

// GetGoVersion returns the version of Go that produced the current binary.
func (b *BuildInfo) GetGoVersion() string {
	return b.goVersion
}
