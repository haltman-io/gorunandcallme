package runtypes

// ExecMode controls how the command is executed.
type ExecMode string

const (
	// ExecModeDirect executes the binary directly with argv, without passing through a shell.
	ExecModeDirect ExecMode = "direct"

	// ExecModeShell executes using a configured shell, passing the raw command string.
	ExecModeShell ExecMode = "shell"
)

type ShellSpec struct {
	// Path is the shell binary path (e.g., /bin/sh, /bin/bash, cmd.exe, powershell.exe).
	Path string

	// Args are the shell arguments BEFORE the command string.
	// Example (sh):   []string{"-c"}
	// Example (bash): []string{"-lc"}
	// Example (cmd):  []string{"/C"}
	// Example (pwsh): []string{"-NoProfile", "-Command"}
	Args []string
}

type CommandSpec struct {
	// Raw is the command as a single string (used mainly for shell mode).
	Raw string

	// Argv is the tokenized form for direct execution.
	Argv []string

	Mode   ExecMode
	Shell  ShellSpec
	Workdir string
	Env     []string
}

type OutputSpec struct {
	AppendTextLine string
	StripANSI      bool
	DropEmptyLines bool
}

type RunRequest struct {
	Command CommandSpec
	Output  OutputSpec
}
