package version

import (
	"fmt"
)

var (
	// GitCommit is filled in by the  compiler / build script
	GitCommit string

	// GitDescribe is filled in by the compiler / build script
	GitDescribe string

	// Number is the base semantic version number of the project
	Number = "0.1.1"

	// PreRelease is the pre-release information for this version
	PreRelease = ""

	// BuildMetadata is the build-metadata of this version
	BuildMetadata = ""

	// BuildTime is the build timestamp in ISO-8601 format
	BuildTime = ""
)

// Info about the version
type Info struct {
	Revision      string
	Number        string
	PreRelease    string
	BuildMetadata string
}

// GetInfo gets the version information
func GetInfo() Info {
	num := Number
	pre := PreRelease
	meta := BuildMetadata

	return Info{
		Revision:      GitCommit,
		Number:        num,
		PreRelease:    pre,
		BuildMetadata: meta,
	}
}

// String returns the semantic version
func (i Info) String() string {
	version := i.Number

	if i.PreRelease != "" {
		version = fmt.Sprintf("%s-%s", version, i.PreRelease)
	}

	if i.BuildMetadata != "" {
		version = fmt.Sprintf("%s+%s", version, i.BuildMetadata)
	}
	return version
}

// FullString returns the full version string optionally including the git revision
func (i Info) FullString(rev bool) string {
	str := i.String()
	if rev && i.Revision != "" {
		return fmt.Sprintf("Damon v%s (%s)", str, i.Revision)
	}
	return fmt.Sprintf("Damon v%s", str)
}
