// Package model holds the shared data types used across portman. Keeping them
// in their own package avoids import cycles between ports, runtime and tray.
package model

// Lang is the detected programming language / runtime of a process.
type Lang string

const (
	Node    Lang = "Node.js"
	Python  Lang = "Python"
	Java    Lang = "Java"
	Ruby    Lang = "Ruby"
	PHP     Lang = "PHP"
	DotNet  Lang = ".NET"
	Go      Lang = "Go"
	Native  Lang = "Native"
	Unknown Lang = "Unknown"
)

// ListenPort describes a single TCP port in the LISTEN state together with the
// process that owns it.
type ListenPort struct {
	Port     int    // listening TCP port
	PID      int32  // owning process id
	ProcName string // process name (e.g. "node", "python3.12")
	Lang     Lang   // detected runtime
	CPU      float64 // CPU percent (since-start average; cheap to read)
	RSS      uint64 // resident memory in bytes
}
