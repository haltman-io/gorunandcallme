package job

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ListOptions struct {
	Limit int // 0 = no limit
}

func CmdList(w io.Writer, st *Store, opt ListOptions) error {
	ids, err := st.ListJobIDs()
	if err != nil {
		return err
	}
	// Show newest first (IDs start with timestamp)
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))

	if opt.Limit > 0 && len(ids) > opt.Limit {
		ids = ids[:opt.Limit]
	}

	for _, id := range ids {
		m, err := st.ReadMeta(id)
		if err != nil {
			fmt.Fprintf(w, "%s  (meta error: %v)\n", id, err)
			continue
		}
		fmt.Fprintf(w, "%s  pid=%d  status=%s  started=%s\n",
			m.ID, m.PID, m.Status, m.StartedAt.Format(time.RFC3339))
	}
	return nil
}

func CmdStatus(w io.Writer, st *Store, jobID string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return errors.New("status: empty job id")
	}
	m, err := st.ReadMeta(jobID)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "id: %s\n", m.ID)
	fmt.Fprintf(w, "pid: %d\n", m.PID)
	fmt.Fprintf(w, "status: %s\n", m.Status)
	fmt.Fprintf(w, "started_at: %s\n", m.StartedAt.Format(time.RFC3339))
	if m.EndedAt != nil {
		fmt.Fprintf(w, "ended_at: %s\n", m.EndedAt.Format(time.RFC3339))
	}
	if m.ExitCode != nil {
		fmt.Fprintf(w, "exit_code: %d\n", *m.ExitCode)
	}
	if m.ErrorText != "" {
		fmt.Fprintf(w, "error: %s\n", m.ErrorText)
	}
	if len(m.RunnerArgs) > 0 {
		fmt.Fprintf(w, "runner_args: %s\n", QuoteArgs(m.RunnerArgs))
	}
	fmt.Fprintf(w, "log: %s\n", m.LogPath)

	return nil
}

type FollowOptions struct {
	Follow   bool          // -f
	Poll     time.Duration // polling interval for -f
	Tail     int           // tail N lines before following; 0 = don't tail
	Stdout   io.Writer
	Ctx      context.Context
	JobStore *Store
}

func CmdFollow(jobID string, opt FollowOptions) error {
	if opt.JobStore == nil {
		return errors.New("follow: JobStore is nil")
	}
	if opt.Stdout == nil {
		opt.Stdout = os.Stdout
	}
	if opt.Ctx == nil {
		opt.Ctx = context.Background()
	}

	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return errors.New("follow: empty job id")
	}

	m, err := opt.JobStore.ReadMeta(jobID)
	if err != nil {
		return err
	}

	if opt.Tail > 0 {
		_ = TailFile(opt.Stdout, m.LogPath, opt.Tail)
	}

	if !opt.Follow {
		// If not following, print entire file best-effort.
		b, err := os.ReadFile(m.LogPath)
		if err != nil {
			return err
		}
		_, _ = opt.Stdout.Write(b)
		return nil
	}

	return FollowFile(opt.Ctx, opt.Stdout, m.LogPath, opt.Poll)
}

func CmdStop(w io.Writer, st *Store, jobID string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return errors.New("stop: empty job id")
	}
	m, err := st.ReadMeta(jobID)
	if err != nil {
		return err
	}
	if m.PID <= 0 {
		return fmt.Errorf("stop: job %s has invalid pid", jobID)
	}

	if err := KillPID(m.PID); err != nil {
		return fmt.Errorf("stop: kill pid %d: %w", m.PID, err)
	}

	now := time.Now().UTC()
	m.Status = StatusStopped
	m.EndedAt = &now
	code := 0
	m.ExitCode = &code

	_ = st.WriteMeta(m)

	fmt.Fprintf(w, "stopped: %s (pid=%d)\n", m.ID, m.PID)
	return nil
}

func CmdPurge(w io.Writer, st *Store, jobID string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return errors.New("purge: empty job id")
	}
	if err := st.DeleteJob(jobID); err != nil {
		return err
	}
	fmt.Fprintf(w, "purged: %s\n", jobID)
	return nil
}

// Optional helper: accept numeric index from "job list" output (if you decide to display indices).
func ParseJobIDOrIndex(st *Store, idOrIndex string) (string, error) {
	idOrIndex = strings.TrimSpace(idOrIndex)
	if idOrIndex == "" {
		return "", errors.New("empty job id")
	}

	// if looks like integer, treat as index on newest-first list
	if n, err := strconv.Atoi(idOrIndex); err == nil {
		if n < 0 {
			return "", errors.New("negative index")
		}
		ids, err := st.ListJobIDs()
		if err != nil {
			return "", err
		}
		sort.Sort(sort.Reverse(sort.StringSlice(ids)))
		if n >= len(ids) {
			return "", fmt.Errorf("index out of range: %d", n)
		}
		return ids[n], nil
	}

	return idOrIndex, nil
}
