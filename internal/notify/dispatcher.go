package notify

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/haltman-io/gorunandcallme/internal/config"
	"github.com/haltman-io/gorunandcallme/internal/event"
)

type Client interface {
	Name() string
	MaxTextChars() int
	MaxAttachBytes() int
	SendText(text string) error
	SendFile(filename string, contentType string, data []byte, caption string) error
}

type Clients struct {
	List []Client
}

type DispatcherOptions struct {
	Clients Clients
	UI      UI
}

type clientWorker struct {
	c  Client
	ch chan job
}

type Dispatcher struct {
	ui      UI
	workers []clientWorker
	wg      sync.WaitGroup

	mu     sync.Mutex
	closed bool
}

type job struct {
	fn func(Client) error
}

func NewDispatcher(o DispatcherOptions) *Dispatcher {
	ui := safeUI(o.UI)

	var workers []clientWorker
	for _, c := range o.Clients.List {
		workers = append(workers, clientWorker{
			c:  c,
			ch: make(chan job, 256),
		})
	}

	d := &Dispatcher{
		ui:      ui,
		workers: workers,
	}

	for _, w := range d.workers {
		w := w
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			d.worker(w)
		}()
	}

	return d
}

func (d *Dispatcher) worker(w clientWorker) {
	for j := range w.ch {
		for attempt := 0; attempt < 5; attempt++ {
			err := j.fn(w.c)
			if err == nil {
				break
			}
			d.ui.Warn("%s notify error: %v (attempt %d)", w.c.Name(), err, attempt+1)
			time.Sleep(time.Duration(500+attempt*500) * time.Millisecond)
		}
	}
}

func (d *Dispatcher) BroadcastText(text string) error {
	if len(d.workers) == 0 {
		return errors.New("no notification clients enabled")
	}
	return d.enqueue(func(c Client) error {
		return c.SendText(text)
	})
}

func (d *Dispatcher) BroadcastFile(filename string, contentType string, data []byte, caption string) error {
	if len(d.workers) == 0 {
		return errors.New("no notification clients enabled")
	}
	fn := filename
	cp := caption
	buf := data
	ct := contentType
	return d.enqueue(func(c Client) error {
		return c.SendFile(fn, ct, buf, cp)
	})
}

func (d *Dispatcher) enqueue(fn func(Client) error) error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return errors.New("dispatcher closed")
	}
	for _, w := range d.workers {
		w.ch <- job{fn: fn}
	}
	d.mu.Unlock()
	return nil
}

func (d *Dispatcher) Close() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	for _, w := range d.workers {
		close(w.ch)
	}
	d.mu.Unlock()
	d.wg.Wait()
}

func HasCallbacks(cliCallbacks []string, cfg *config.Config) bool {
	if len(cliCallbacks) > 0 {
		return true
	}
	return len(cfg.Notify.Callbacks) > 0
}

func WantsLifecycle(notifyOn []string, item string) bool {
	for _, v := range notifyOn {
		if v == item {
			return true
		}
	}
	return false
}

func BuildClients(httpc *http.Client, cfg *config.Config) (Clients, error) {
	cbs := cfg.Notify.Callbacks
	if len(cbs) == 0 {
		return Clients{}, nil
	}

	expanded := []string{}
	for _, c := range cbs {
		if c == "all" {
			expanded = append(expanded, "discord", "slack", "telegram", "webhook")
		} else {
			expanded = append(expanded, c)
		}
	}

	var out []Client

	for _, cb := range expanded {
		switch cb {
		case "discord":
			if cfg.Discord.WebhookURL == "" {
				return Clients{}, errors.New("discord enabled but webhook_url is empty")
			}
			c, err := NewDiscordClient(httpc, cfg.Discord.WebhookURL, cfg.Notify.Attach.PartMaxBytes["discord"])
			if err != nil {
				return Clients{}, err
			}
			out = append(out, c)
		case "slack":
			// slack can work with webhook only; attachments require bot token+channel.
			if cfg.Slack.WebhookURL == "" && cfg.Slack.BotToken == "" {
				return Clients{}, errors.New("slack enabled but webhook_url/bot_token not set")
			}
			c, err := NewSlackClient(httpc, cfg.Slack.WebhookURL, cfg.Slack.BotToken, cfg.Slack.Channel)
			if err != nil {
				return Clients{}, err
			}
			out = append(out, c)
		case "telegram":
			if cfg.Telegram.BotToken == "" || cfg.Telegram.ChatID == "" {
				return Clients{}, errors.New("telegram enabled but bot_token/chat_id not set")
			}
			c, err := NewTelegramClient(httpc, cfg.Telegram.BotToken, cfg.Telegram.ChatID, cfg.Telegram.ParseMode)
			if err != nil {
				return Clients{}, err
			}
			out = append(out, c)
		case "webhook":
			if cfg.Webhook.URL == "" {
				return Clients{}, errors.New("webhook enabled but url is empty")
			}
			c, err := NewWebhookClient(httpc, cfg.Webhook.URL, cfg.Webhook.Headers)
			if err != nil {
				return Clients{}, err
			}
			out = append(out, c)
		default:
			return Clients{}, errors.New("unknown callback: " + cb)
		}
	}

	return Clients{List: out}, nil
}

// EventSink is a thin wrapper around JSONL event writer.
type EventSink struct {
	w  *event.Writer
	ui UI
}

func NewEventSink(path string, ui UI) *EventSink {
	u := safeUI(ui)
	w, err := event.New(path)
	if err != nil {
		u.Warn("event-output disabled: %v", err)
		return &EventSink{w: nil, ui: u}
	}
	return &EventSink{w: w, ui: u}
}

func (e *EventSink) Write(ev event.Event) error {
	if e == nil || e.w == nil {
		return nil
	}
	return e.w.Write(ev)
}

func (e *EventSink) Close() {
	if e == nil || e.w == nil {
		return
	}
	e.w.Close()
}

type noopUI struct{}

func (noopUI) Info(string, ...any)     {}
func (noopUI) Warn(string, ...any)     {}
func (noopUI) Error(string, ...any)    {}
func (noopUI) Verbosef(string, ...any) {}
func (noopUI) Debugf(string, ...any)   {}

func safeUI(ui UI) UI {
	if ui == nil {
		return noopUI{}
	}
	return ui
}
