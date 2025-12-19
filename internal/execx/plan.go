package execx

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type PlanOptions struct {
	ExecMode   string
	Shell      string
	CommandStr string
	Args       []string
	CWD        string
	EnvPairs   []string
	StdinMode  string
	NoColor    bool
}

type Plan struct {
	mode      Mode
	command   string
	args      []string
	shellPath string
	shellArgs []string
	workdir   string
	env       []string
}

func BuildPlan(opt PlanOptions) (*Plan, error) {
	modeStr := opt.ExecMode
	if strings.TrimSpace(modeStr) == "" {
		modeStr = string(ModeDirect)
	}
	mode, err := ParseMode(modeStr)
	if err != nil {
		return nil, err
	}

	p := &Plan{
		mode:    mode,
		workdir: opt.CWD,
	}

	switch mode {
	case ModeDirect:
		if len(opt.Args) == 0 {
			return nil, errors.New("exec-mode=direct requires command args")
		}
		p.command = opt.Args[0]
		if len(opt.Args) > 1 {
			p.args = opt.Args[1:]
		}
	default:
		if strings.TrimSpace(opt.CommandStr) == "" {
			return nil, errors.New("shell execution requires --command")
		}
		path, sargs, err := shellSpec(mode, opt.Shell)
		if err != nil {
			return nil, err
		}
		p.command = opt.CommandStr
		p.shellPath = path
		p.shellArgs = sargs
	}

	env := os.Environ()
	if len(opt.EnvPairs) > 0 {
		env = append(env, opt.EnvPairs...)
	}
	p.env = env

	return p, nil
}

func (p *Plan) Describe() string {
	if p == nil {
		return ""
	}
	if p.mode == ModeDirect {
		return strings.Join(append([]string{p.command}, p.args...), " ")
	}
	return p.command
}

func (p *Plan) buildCmd() (*exec.Cmd, error) {
	if p == nil {
		return nil, errors.New("nil plan")
	}

	var cmd *exec.Cmd
	if p.mode == ModeDirect {
		cmd = exec.Command(p.command, p.args...)
	} else {
		args := append([]string{}, p.shellArgs...)
		args = append(args, p.command)
		cmd = exec.Command(p.shellPath, args...)
	}

	if p.workdir != "" {
		cmd.Dir = p.workdir
	}
	if len(p.env) > 0 {
		cmd.Env = p.env
	}
	return cmd, nil
}

func shellSpec(mode Mode, override string) (string, []string, error) {
	path := strings.TrimSpace(override)
	switch mode {
	case ModeShell:
		if path == "" {
			if runtime.GOOS == "windows" {
				path = "cmd.exe"
			} else {
				path = "/bin/sh"
			}
		}
		if runtime.GOOS == "windows" {
			return path, []string{"/C"}, nil
		}
		return path, []string{"-c"}, nil
	case ModeBash:
		if path == "" {
			path = "bash"
		}
		return path, []string{"-lc"}, nil
	case ModeZsh:
		if path == "" {
			path = "zsh"
		}
		return path, []string{"-lc"}, nil
	case ModePwsh:
		if path == "" {
			path = "pwsh"
		}
		return path, []string{"-NoProfile", "-Command"}, nil
	case ModeCmd:
		if path == "" {
			path = "cmd.exe"
		}
		return path, []string{"/C"}, nil
	case ModeCustom:
		if path == "" {
			return "", nil, errors.New("exec-mode=custom requires --shell <path>")
		}
		return path, []string{"-c"}, nil
	default:
		return "", nil, errors.New("unsupported exec mode")
	}
}
