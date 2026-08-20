// Package model holds the shared data types used across portman. Keeping them
// in their own package avoids import cycles between ports, runtime and tray.
package model

// Lang is the detected programming language / runtime of a process.
type Lang string

const (
	Node    Lang = "Node.js"
	Bun     Lang = "Bun"
	Deno    Lang = "Deno"
	Python  Lang = "Python"
	Java    Lang = "Java"
	Ruby    Lang = "Ruby"
	PHP     Lang = "PHP"
	DotNet  Lang = ".NET"
	Go      Lang = "Go"
	Rust    Lang = "Rust"
	Elixir  Lang = "Elixir"
	Ollama  Lang = "Ollama"
	Native  Lang = "Native"
	Unknown Lang = "Unknown"
)

// ListenPort describes a single TCP port in the LISTEN state together with the
// process that owns it.
type ListenPort struct {
	Port      int     // listening TCP port
	PID       int32   // owning process id
	ProcName  string  // process name (e.g. "node", "python3.12")
	Exe       string  // absolute path to the executable (may be empty)
	Lang      Lang    // detected runtime
	Project   string  // detected project name (from package.json/go.mod/...)
	Framework string  // detected framework (e.g. "Next.js", "Vite")
	Cwd       string  // process working directory (powers "Reveal")
	CreatedMs int64   // process start time, ms since epoch (0 if unknown)
	Alive     bool    // port currently accepts TCP connections (health probe)
	CPU       float64 // CPU percent (since-start average; cheap to read)
	RSS       uint64  // resident memory in bytes
}
