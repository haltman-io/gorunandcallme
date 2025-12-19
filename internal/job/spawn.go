package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type SpawnOptions struct {
	// The *runner* executable (gorunandcallme) to spawn.
	// Usually os.Executable() from main.
	RunnerPath string

	// Args passed to the runner worker instance. Example:
	// ["--worker", "--command", "subfinder -d example.com", "--notify-each", "10s", ...]
	RunnerArgs []string

	// Working directory for the worker.
	Workdir string

	// Store persists metadata and the log path.
	Store *Store
}

type SpawnResult struct {
	JobID   string
	PID     int
	Meta    Meta
	LogPath string
}

func SpawnBackground(ctx context.Context, opt SpawnOptions) (SpawnResult, error) {
	if opt.Store == nil {
		return SpawnResult{}, errors.New("spawn: Store is nil")
	}
	if strings.TrimSpace(opt.RunnerPath) == "" {
		return SpawnResult{}, errors.New("spawn: RunnerPath is empty")
	}

	jobID, err := newJobID()
	if err != nil {
		return SpawnResult{}, fmt.Errorf("spawn: job id: %w", err)
	}

	if err := opt.Store.CreateJobDirs(jobID); err != nil {
		return SpawnResult{}, fmt.Errorf("spawn: create dirs: %w", err)
	}

	logPath := opt.Store.LogPath(jobID)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return SpawnResult{}, fmt.Errorf("spawn: open log: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	// Detach stdin from the terminal.
	devNullPath := devNull()
	devNull, _ := os.Open(devNullPath) // best-effort
	defer func() {
		if devNull != nil {
			_ = devNull.Close()
		}
	}()

	cmd := exec.CommandContext(ctx, opt.RunnerPath, opt.RunnerArgs...)
	if strings.TrimSpace(opt.Workdir) != "" {
		cmd.Dir = opt.Workdir
	}

	// Redirect output to log file.
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if devNull != nil {
		cmd.Stdin = devNull
	}

	// Detach / run in background.
	daemonize(cmd)

	// Persist initial meta before start.
	meta := Meta{
		ID:         jobID,
		PID:        0,
		StartedAt:  time.Now().UTC(),
		RunnerArgs: append([]string{}, opt.RunnerArgs...),
		Workdir:    cmd.Dir,
		LogPath:    logPath,
		Status:     StatusStarting,
	}

	if err := opt.Store.WriteMeta(meta); err != nil {
		return SpawnResult{}, fmt.Errorf("spawn: write meta (starting): %w", err)
	}

	if err := cmd.Start(); err != nil {
		meta.Status = StatusFailed
		meta.ErrorText = err.Error()
		_ = opt.Store.WriteMeta(meta)
		return SpawnResult{}, fmt.Errorf("spawn: start: %w", err)
	}

	meta.PID = cmd.Process.Pid
	meta.Status = StatusRunning

	if err := opt.Store.WriteMeta(meta); err != nil {
		return SpawnResult{}, fmt.Errorf("spawn: write meta (running): %w", err)
	}

	// Do NOT wait. We return immediately to free the terminal.
	return SpawnResult{
		JobID:   jobID,
		PID:     meta.PID,
		Meta:    meta,
		LogPath: logPath,
	}, nil
}

func newJobID() (string, error) {
	// 8 random bytes = 16 hex chars. Good enough for job IDs.
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// add time prefix to help sorting by creation time (lexicographically),
	// while keeping uniqueness from random suffix.
	ts := time.Now().UTC().Format("20060102T150405Z")
	return ts + "-" + hex.EncodeToString(b), nil
}

func devNull() string {
	if runtime.GOOS == "windows" {
		return "NUL"
	}
	return "/dev/null"
}

// Small helper for printing command lines safely in logs/UI.
func QuoteArgs(args []string) string {
	// Minimal quoting: if contains spaces or quotes, wrap with double quotes and escape quotes.
	var out []string
	for _, a := range args {
		if strings.ContainsAny(a, " \t\r\n\"") {
			a = strings.ReplaceAll(a, `"`, `\"`)
			out = append(out, `"`+a+`"`)
		} else {
			out = append(out, a)
		}
	}
	return strings.Join(out, " ")
}

// TailFile prints the last N lines from a file (best-effort, no heavy indexing).
func TailFile(w io.Writer, path string, maxLines int) error {
	if maxLines <= 0 {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	// handle final newline producing empty last element
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	start := 0
	if len(lines) > maxLines {
		start = len(lines) - maxLines
	}
	for _, ln := range lines[start:] {
		fmt.Fprintln(w, ln)
	}
	return nil
}

// FollowFile streams appended content from a file (polling).
func FollowFile(ctx context.Context, w io.Writer, path string, poll time.Duration) error {
	if poll <= 0 {
		poll = 500 * time.Millisecond
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// Seek to end initially (like tail -f, default behavior).
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := f.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				time.Sleep(poll)
				continue
			}
			return err
		}
	}
}

// Ensures store base dir exists in a portable way when caller doesn't set it.
func EnsureDefaultStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return NewStore(filepath.Join(home, DefaultAppDirName))
}
