package execx

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/haltman-io/gorunandcallme/internal/event"
)

type SchedulerUI interface {
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
}

type LineHook interface {
	OnLine(stream string, line string)
}

type EventSink interface {
	Write(ev event.Event) error
}

type SchedulerOptions struct {
	Threads    int
	UI         SchedulerUI
	EventSink  EventSink
	NoTTY      bool
	StripANSI  bool
	StripProg  bool
	NotifyHook LineHook
	OutputFile string
	OutputMode string
}

type Scheduler struct {
	opt SchedulerOptions
}

func NewScheduler(opt SchedulerOptions) *Scheduler {
	if opt.Threads <= 0 {
		opt.Threads = 1
	}
	return &Scheduler{opt: opt}
}

func (s *Scheduler) Run(plan *Plan) (int, error) {
	if plan == nil {
		return 0, errors.New("nil plan")
	}

	cmd, err := plan.buildCmd()
	if err != nil {
		return 0, err
	}

	var outFile *os.File
	if strings.TrimSpace(s.opt.OutputFile) != "" {
		outFile, err = os.Create(s.opt.OutputFile)
		if err != nil {
			return 0, err
		}
		defer outFile.Close()
	}

	onLine := func(line string) {
		if s.opt.NotifyHook != nil {
			s.opt.NotifyHook.OnLine("stdout", line)
		}
		if outFile != nil {
			_, _ = outFile.WriteString(line + "\n")
		}
		if s.opt.EventSink != nil {
			_ = s.opt.EventSink.Write(event.Event{
				Type:    "line",
				Stream:  "stdout",
				Message: line,
			})
		}
	}

	exitCode, err := RunCommand(context.Background(), cmd, RunnerOptions{
		MirrorToTTY:   !s.opt.NoTTY,
		StripANSI:     s.opt.StripANSI,
		StripProgress: s.opt.StripProg,
	}, onLine)
	if err != nil {
		return exitCode, err
	}
	return exitCode, nil
}
