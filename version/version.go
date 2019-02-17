package version

var (
	GitCommit     string
	Version       = "1.0.0"
	VersionSuffix = "dev"
)

// Humanly readable version.
func HumanVersion() string {
	version := Version

	if VersionSuffix != "" {
		version += "-" + VersionSuffix
	}

	if GitCommit != "" {
		version += " (" + GitCommit + ")"
	}

	return version
}
