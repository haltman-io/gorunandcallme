package notify

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/haltman-io/gorunandcallme/internal/config"
	"github.com/haltman-io/gorunandcallme/internal/util"
)

type AggregatorOptions struct {
	Config   config.NotifyConfig
	Redactor *Redactor
	Filters  *Filters
	Alerts   *Alerts
	UI       UI
	Dispatch *Dispatcher
}

type Aggregator struct {
	cfg   config.NotifyConfig
	ui    UI
	disp  *Dispatcher
	red   *Redactor
	filt  *Filters
	alert *Alerts

	mu       sync.Mutex
	lines    []string
	context  []string
	lastTick time.Time
	ticker   *time.Ticker
	stop     chan struct{}
}

func NewAggregator(o AggregatorOptions) (*Aggregator, error) {
	a := &Aggregator{
		cfg:    o.Config,
		ui:     o.UI,
		disp:   o.Dispatch,
		red:    o.Redactor,
		filt:   o.Filters,
		alert:  o.Alerts,
		stop:   make(chan struct{}),
		lines:  nil,
		context: nil,
	}

	if a.cfg.NotifyEach != "" {
		d, err := util.ParseExtendedDuration(a.cfg.NotifyEach)
		if err != nil {
			return nil, err
		}
		if d > 0 {
			a.ticker = time.NewTicker(d)
			go a.loop()
		}
	}
	return a, nil
}

func (a *Aggregator) loop() {
	for {
		select {
		case <-a.stop:
			return
		case <-a.ticker.C:
			a.FlushAll("tick")
		}
	}
}

func (a *Aggregator) Close() {
	if a == nil {
		return
	}
	if a.ticker != nil {
		a.ticker.Stop()
	}
	// prevent panic if Close is called twice
	select {
	case <-a.stop:
		return
	default:
		close(a.stop)
	}
}

func (a *Aggregator) OnLine(stream string, line string) {
	if a == nil {
		return
	}

	// Apply filters and redaction for notification pipeline.
	// This intentionally does not affect terminal output.
	if a.red != nil {
		line = a.red.Apply(line)
	}
	if a.filt != nil && !a.filt.Allow(line) {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.lines = append(a.lines, line)
	a.context = append(a.context, line)
	if a.alert != nil && a.alert.ContextLines() > 0 && len(a.context) > a.alert.ContextLines() {
		a.context = a.context[len(a.context)-a.alert.ContextLines():]
	}

	// Immediate alert on match
	if a.alert != nil && a.alert.Match(line) {
		ctx := append([]string{}, a.context...)
		go a.sendAlert(line, ctx)
	}
}

func (a *Aggregator) sendAlert(matched string, ctx []string) {
	title := fmt.Sprintf("ALERT: matched output pattern")
	body := "Matched:\n" + matched + "\n\nContext:\n" + JoinLines(ctx)
	a.sendText(title, body)
}

func (a *Aggregator) FlushAll(reason string) {
	if a == nil || a.disp == nil {
		return
	}

	a.mu.Lock()
	lines := append([]string{}, a.lines...)
	a.lines = nil
	a.lastTick = time.Now()
	a.mu.Unlock()

	if len(lines) == 0 {
		return
	}

	mode := strings.ToLower(a.cfg.Mode)
	switch mode {
	case "summary":
		txt := Summary(lines, 30)
		a.sendText("Output summary", txt)
		return
	}

	// Select lines for text mode
	selected := lines
	switch strings.ToLower(a.cfg.Text.Select) {
	case "head":
		selected = HeadLines(selected, a.cfg.Text.HeadLines)
	case "tail":
		selected = TailLines(selected, a.cfg.Text.TailLines)
	default:
		// all
	}

	text := JoinLines(selected)

	switch mode {
	case "text-only":
		a.sendText("Output batch", text)
		return
	case "attach-only":
		if !a.cfg.Attach.Enabled {
			_ = a.disp.BroadcastText(WrapCodeBlockMarkdown(text))
			return
		}
		a.sendAttach("Output batch", lines)
		return
	default: // auto
		// Heuristic: if too long, attach; else text.
		if len(text) > 3500 && a.cfg.Attach.Enabled {
			a.sendAttach("Output batch", lines)
			return
		}
		a.sendText("Output batch", text)
		return
	}
}

func (a *Aggregator) SendLifecycle(state string, fullCmd string, details string) {
	title := fmt.Sprintf("Job %s", state)
	msg := fmt.Sprintf("%s `%s`\n%s", strings.Title(state), fullCmd, details)
	a.sendText(title, msg)
}

func (a *Aggregator) sendText(title string, text string) {
	if a == nil || a.disp == nil {
		return
	}
	payload := title + "\n" + WrapCodeBlockMarkdown(text)
	_ = a.disp.BroadcastText(payload)
}

func (a *Aggregator) sendAttach(title string, lines []string) {
	if a == nil || a.disp == nil {
		return
	}

	// Build per-platform parts using each client's max attachment bytes if needed.
	// Dispatcher broadcasts: we must split based on worst-case. Use webhook default if unknown.
	// User can override per platform via config/flag.
	max := a.cfg.Attach.PartMaxBytes["webhook"]
	if max <= 0 {
		max = 10000000
	}
	if v, ok := a.cfg.Attach.PartMaxBytes["discord"]; ok && v > 0 && v < max {
		max = v
	}
	if v, ok := a.cfg.Attach.PartMaxBytes["telegram"]; ok && v > 0 && v < max {
		max = v
	}

	switch strings.ToLower(a.cfg.Attach.SplitMode) {
	case "tail":
		lines = TailLines(lines, a.cfg.Attach.TailLines)
		data := []byte(JoinLines(lines) + "\n")
		_ = a.disp.BroadcastFile("output.log", "text/plain", data, title)
		return
	default: // split
		parts := BuildAttachmentParts(lines, max)
		for i, p := range parts {
			caption := fmt.Sprintf("%s (part %d/%d)", title, i+1, len(parts))
			_ = a.disp.BroadcastFile(fmt.Sprintf("output.part.%03d.log", i+1), "text/plain", p, caption)
		}
		return
	}
}
