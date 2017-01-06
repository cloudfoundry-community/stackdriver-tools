package version

const Name = "cf-stackdriver-nozzle"

var release string

func init() {
	// release is set by the linker on published builds
	if release == "" {
		release = "dev"
	}
}

func Release() string {
	return release
}

func UserAgent() string {
	return Name + "/" + release
}
