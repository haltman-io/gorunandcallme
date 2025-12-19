package execx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/haltman-io/gorunandcallme/internal/util"
)

// NOTE:
// This file must NOT import "internal/app".
// Importing app from execx creates an import cycle:
// app -> execx -> app (cycle not allowed).
//
// Any UI/logging integration is done via interfaces (or plain io.Writers).

// RunnerUI is an optional logger interface.
// It is intentionally minimal and decoupled from internal/app.
type RunnerUI interface {
	Verbosef(format string, args ...any)
	Debugf(format string, args ...any)
}

// RunnerOptions controls how output is handled.
type RunnerOptions struct {
	UI RunnerUI

	// MirrorToTTY controls if child output is mirrored to current process stdout/stderr.
	// Notification hooks are handled elsewhere by the scheduler.
	MirrorToTTY bool

	// Strip ANSI escape sequences from child output.
	StripANSI bool

	// Strip carriage-return based progress lines (e.g., spinners).
	StripProgress bool

	// Prefix added at the beginning of each line (example: "[abc]").
	AppendTextLine string

	// Drop empty lines after sanitization.
	DropEmpty bool

	// Scanner max token size for very long lines (defaults to 8 MiB).
	MaxLineBytes int
}

// LineHandler receives sanitized lines in real-time.
type LineHandler func(line string)

// RunCommand executes cmd and streams stdout/stderr as lines.
// It returns the process exit code (even when non-zero) and a hard error for start/wait issues.
func RunCommand(ctx context.Context, cmd *exec.Cmd, opt RunnerOptions, onLine LineHandler) (int, error) {
	if cmd == nil {
		return 0, errors.New("nil cmd")
	}
	if onLine == nil {
		onLine = func(string) {}
	}
	if opt.MaxLineBytes <= 0 {
		opt.MaxLineBytes = 8 * 1024 * 1024
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("stderr pipe: %w", err)
	}

	// If mirroring to tty is desired, tee both streams to stdout/stderr.
	// But we still need line-based processing, so we do it by writing in the read loop.
	var ttyOut io.Writer = io.Discard
	var ttyErr io.Writer = io.Discard
	if opt.MirrorToTTY {
		ttyOut = os.Stdout
		ttyErr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	readStream := func(r io.Reader, mirror io.Writer) {
		defer wg.Done()

		sc := bufio.NewScanner(r)
		// Raise scanner limits for big lines
		buf := make([]byte, 64*1024)
		sc.Buffer(buf, opt.MaxLineBytes)

		for sc.Scan() {
			line := sc.Text()

			// Optionally mirror raw-ish output (still line-based here).
			// This keeps terminal usable without embedding ANSI in notifications.
			if opt.MirrorToTTY {
				_, _ = fmt.Fprintln(mirror, line)
			}

			// Sanitize for downstream (notify/output file/etc).
			if opt.StripProgress {
				line = util.StripProgress(line)
			}
			if opt.StripANSI {
				line = util.StripANSI(line)
			}
			if opt.AppendTextLine != "" {
				line = opt.AppendTextLine + line
			}
			if opt.DropEmpty && strings.TrimSpace(line) == "" {
				continue
			}

			onLine(line)
		}
	}

	go readStream(stdout, ttyOut)
	go readStream(stderr, ttyErr)

	waitErr := cmd.Wait()
	wg.Wait()

	exitCode := exitCodeFromWait(waitErr)

	// Non-zero exit is not a "hard" error for tooling.
	// Only propagate errors that are not normal exit-status.
	if waitErr != nil && !isExitStatus(waitErr) {
		return exitCode, fmt.Errorf("wait: %w", waitErr)
	}

	return exitCode, nil
}

func isExitStatus(err error) bool {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return true
	}
	return false
}

func exitCodeFromWait(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		// Unix
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			return ws.ExitStatus()
		}
		// Fallback
		return 1
	}
	return 1
}

// Optional helper used by schedulers: run with timeout.
func RunCommandWithTimeout(cmd *exec.Cmd, opt RunnerOptions, onLine LineHandler, timeout time.Duration) (int, error) {
	if timeout <= 0 {
		return RunCommand(context.Background(), cmd, opt, onLine)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return RunCommand(ctx, cmd, opt, onLine)
}
