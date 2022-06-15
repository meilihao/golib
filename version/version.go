// from https://github.com/pingcap/tidb/blob/master/util/versioninfo/versioninfo.go
package version

import (
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var (
	appName        = ""
	component      = ""
	gitTag         = ""
	gitBranch      = ""
	gitHash        = ""
	buildTimestamp = ""

	gitStateDirtySuffix = "-dirty"

	versionReg = regexp.MustCompile("^[0-9a-z]{7}") // match git hash
)

// BuildInfo describes the compile time information.
type BuildInfo struct {
	AppName   string `json:"app_name"`
	Component string `json:"component"`
	// Version is the current semver by custom, defualt please use git tag.
	Version string `json:"version"`
	Arch    string `json:"arch"`
	GitTag  string `json:"git_tag,omitempty"`
	// GitBranch is the brach of the git tree.
	GitBranch string `json:"git_brach,omitempty"`
	// GitHash is the git sha1.
	GitHash string `json:"git_hash,omitempty"`
	// GitState is the state of the git tree
	GitState string `json:"git_state,omitempty"`
	// BuildTimestate build time
	BuildTimestamp time.Time `json:"build_timestamp,omitempty"`
	// GoVersion is the version of the Go compiler used.
	GoVersion string `json:"go_version,omitempty"`
}

// Get returns build info
func NewBuildInfo() *BuildInfo {
	t, _ := time.ParseInLocation("2006-01-02 15:04:05Z07:00", buildTimestamp, time.UTC)

	i := BuildInfo{
		AppName:        appName,
		Component:      component,
		Version:        "",
		Arch:           runtime.GOARCH,
		GitTag:         gitTag,
		GitBranch:      gitBranch,
		GitHash:        gitHash,
		GitState:       "clean",
		BuildTimestamp: t,
		GoVersion:      runtime.Version(),
	}

	// 354ee8b-dirty
	if strings.HasSuffix(i.GitTag, gitStateDirtySuffix) {
		i.GitTag = strings.TrimSuffix(i.GitTag, gitStateDirtySuffix)
		i.GitState = "dirty"
	}
	if i.Version == "" {
		i.Version = i.GitTag
	}

	return &i
}

func (i *BuildInfo) String() string {
	data, _ := json.Marshal(i)

	return string(data)
}

func (i *BuildInfo) Fullname() string {
	if i.AppName == "" {
		return fmt.Sprintf("%s-%s-%s-%s", i.Component, i.Version, i.Arch, i.BuildTimestamp.Format("20060102150405"))
	}

	return fmt.Sprintf("%s.%s-%s-%s-%s", i.AppName, i.Component, i.Version, i.Arch, i.BuildTimestamp.Format("20060102150405"))
}
