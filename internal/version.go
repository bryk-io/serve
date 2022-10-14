package internal

// CoreVersion defines the semantic version of the build. If not available
// a short version of the commit hash will be used.
var CoreVersion string

// BuildCode provides the commit identifier used to build the binary.
var BuildCode string

// BuildTimestamp provides the UNIX timestamp of the build.
var BuildTimestamp string
