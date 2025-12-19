package execx

import (
	"errors"
	"fmt"
	"strings"
)

type Mode string

const (
	ModeDirect Mode = "direct"
	ModeShell  Mode = "shell"
	ModeBash   Mode = "bash"
	ModeZsh    Mode = "zsh"
	ModePwsh   Mode = "pwsh"
	ModeCmd    Mode = "cmd"
	ModeCustom Mode = "custom"
)

func ParseMode(s string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "direct":
		return ModeDirect, nil
	case "shell":
		return ModeShell, nil
	case "bash":
		return ModeBash, nil
	case "zsh":
		return ModeZsh, nil
	case "pwsh":
		return ModePwsh, nil
	case "cmd":
		return ModeCmd, nil
	case "custom":
		return ModeCustom, nil
	default:
		return "", fmt.Errorf("invalid exec mode: %s", s)
	}
}

func ValidateModeInput(mode Mode, commandStr string, args []string, shellPath string) error {
	if mode == ModeDirect {
		if len(args) == 0 {
			return errors.New("exec-mode=direct requires args after '--': gorunandcallme -- <cmd> [args...]")
		}
		return nil
	}
	// shell-based
	if commandStr == "" {
		return errors.New("shell execution requires --command \"...\"")
	}
	if mode == ModeCustom && shellPath == "" {
		return errors.New("exec-mode=custom requires --shell <path>")
	}
	return nil
}
