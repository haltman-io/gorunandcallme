package job

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DefaultAppDirName = ".gorunandcallme"
	DefaultJobsDir    = "jobs"
	MetaFileName      = "meta.json"
	LogFileName       = "output.log"
)

type Status string

const (
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusFinished Status = "finished"
	StatusFailed   Status = "failed"
	StatusStopped  Status = "stopped"
)

type Meta struct {
	ID        string    `json:"id"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`

	// "Runner" is your own tool (gorunandcallme) command line used for the background worker,
	// not the target security tool command itself.
	RunnerArgs []string `json:"runner_args,omitempty"`

	Workdir string `json:"workdir,omitempty"`
	LogPath string `json:"log_path"`

	Status    Status     `json:"status"`
	ExitCode  *int       `json:"exit_code,omitempty"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	ErrorText string     `json:"error_text,omitempty"`
}

type Store struct {
	BaseDir string
}

func NewStore(baseDir string) (*Store, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		baseDir = filepath.Join(home, DefaultAppDirName)
	}
	if err := os.MkdirAll(filepath.Join(baseDir, DefaultJobsDir), 0o700); err != nil {
		return nil, fmt.Errorf("create jobs dir: %w", err)
	}
	return &Store{BaseDir: baseDir}, nil
}

func (s *Store) JobsDir() string {
	return filepath.Join(s.BaseDir, DefaultJobsDir)
}

func (s *Store) JobDir(id string) string {
	return filepath.Join(s.JobsDir(), id)
}

func (s *Store) MetaPath(id string) string {
	return filepath.Join(s.JobDir(id), MetaFileName)
}

func (s *Store) LogPath(id string) string {
	return filepath.Join(s.JobDir(id), LogFileName)
}

func (s *Store) CreateJobDirs(id string) error {
	dir := s.JobDir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create job dir: %w", err)
	}
	return nil
}

func (s *Store) WriteMeta(m Meta) error {
	if m.ID == "" {
		return errors.New("meta: empty ID")
	}
	path := s.MetaPath(m.ID)

	tmp := path + ".tmp"
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write tmp meta: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename meta: %w", err)
	}
	return nil
}

func (s *Store) ReadMeta(id string) (Meta, error) {
	path := s.MetaPath(id)
	b, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, fmt.Errorf("read meta: %w", err)
	}
	var m Meta
	if err := json.Unmarshal(b, &m); err != nil {
		return Meta{}, fmt.Errorf("unmarshal meta: %w", err)
	}
	return m, nil
}

func (s *Store) ListJobIDs() ([]string, error) {
	entries, err := os.ReadDir(s.JobsDir())
	if err != nil {
		return nil, fmt.Errorf("readdir jobs: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.TrimSpace(name) == "" {
			continue
		}
		ids = append(ids, name)
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *Store) DeleteJob(id string) error {
	return os.RemoveAll(s.JobDir(id))
}
