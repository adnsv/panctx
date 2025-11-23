package main

import (
	"runtime/debug"
)

var app_ver string = ""

// app_version returns the application version string. It attempts to retrieve the version
// from build information (for go install), falls back to ldflags-injected version, or
// returns "#UNAVAILABLE" if no version information is available.
func app_version() string {
	v, ok := debug.ReadBuildInfo()
	if ok && v.Main.Version != "(devel)" {
		// installed with go install
		return v.Main.Version
	} else if app_ver != "" {
		// built with ld-flags
		return app_ver
	} else {
		return "#UNAVAILABLE"
	}
}
