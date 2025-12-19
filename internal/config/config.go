package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	StateDir    string          `yaml:"state_dir"`
	Transport   TransportConfig  `yaml:"transport"`
	Notify      NotifyConfig     `yaml:"notify"`
	Discord     DiscordConfig    `yaml:"discord"`
	Slack       SlackConfig      `yaml:"slack"`
	Telegram    TelegramConfig   `yaml:"telegram"`
	Webhook     WebhookConfig    `yaml:"webhook"`
	EventOutput string          `yaml:"event_output"`
	Profiles    map[string]*ProfileConfig `yaml:"profiles"`
}

type ProfileConfig struct {
	Notify   *NotifyConfig   `yaml:"notify"`
	Discord  *DiscordConfig  `yaml:"discord"`
	Slack    *SlackConfig    `yaml:"slack"`
	Telegram *TelegramConfig `yaml:"telegram"`
	Webhook  *WebhookConfig  `yaml:"webhook"`
}

type TransportConfig struct {
	Proxy       string `yaml:"proxy"`
	ProxyAuth   string `yaml:"proxy_auth"`
	NoProxyEnv  bool   `yaml:"no_proxy"`
	InsecureTLS bool   `yaml:"insecure"`
}

type NotifyConfig struct {
	Callbacks     []string `yaml:"callbacks"`
	NotifyEach    string   `yaml:"notify_each"`
	NotifyOn      []string `yaml:"notify_on"` // start, finish
	Mode          string   `yaml:"mode"`      // text-only | attach-only | auto | summary
	StripANSI     string   `yaml:"strip_ansi"`
	StripProgress string   `yaml:"strip_progress"`

	Text     NotifyTextConfig   `yaml:"text"`
	Attach   NotifyAttachConfig `yaml:"attach"`
	Filters  NotifyFilterConfig `yaml:"filters"`
	Redaction RedactionConfig   `yaml:"redaction"`
	Alerts   AlertsConfig       `yaml:"alerts"`
}

type NotifyTextConfig struct {
	Select    string `yaml:"select"` // all | head | tail
	HeadLines int    `yaml:"head_lines"`
	TailLines int    `yaml:"tail_lines"`
}

type NotifyAttachConfig struct {
	Enabled      bool           `yaml:"enabled"`
	SplitMode    string         `yaml:"split_mode"` // split|tail
	TailLines    int            `yaml:"tail_lines"`
	PartMaxBytes map[string]int `yaml:"part_max_bytes"`
}

type NotifyFilterConfig struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

type RedactionConfig struct {
	Defaults bool     `yaml:"defaults"`
	Patterns []string `yaml:"patterns"`
	File     string   `yaml:"file"`
}

type AlertsConfig struct {
	Patterns            []string `yaml:"patterns"`
	IncludeContextLines int      `yaml:"include_context_lines"`
}

type DiscordConfig struct {
	WebhookURL string `yaml:"webhook_url"`
}

type SlackConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	BotToken   string `yaml:"bot_token"`
	Channel    string `yaml:"channel"`
}

type TelegramConfig struct {
	BotToken   string `yaml:"bot_token"`
	ChatID     string `yaml:"chat_id"`
	ParseMode  string `yaml:"parse_mode"` // MarkdownV2
}

type WebhookConfig struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
}

type CLIOverrides struct {
	DiscordWebhookURL string
	SlackWebhookURL   string
	SlackBotToken     string
	SlackChannel      string
	TelegramBotToken  string
	TelegramChatID    string
	WebhookURL        string
	WebhookHeaders    []string // KEY=VALUE
}

type LoadOptions struct {
	ConfigPath string
	Profile    string
	StateDir   string
	CLI        CLIOverrides
}

func LoadMerged(o LoadOptions) (*Config, error) {
	cfg := DefaultConfig()

	// load YAML file if provided
	if o.ConfigPath != "" {
		raw, err := os.ReadFile(o.ConfigPath)
		if err != nil {
			return nil, err
		}
		var fromFile Config
		if err := yaml.Unmarshal(raw, &fromFile); err != nil {
			return nil, err
		}
		*cfg = mergeConfig(*cfg, fromFile)
	}

	// profile overlay
	if len(cfg.Profiles) > 0 && o.Profile != "" {
		p := cfg.Profiles[o.Profile]
		cfg = MergeBaseWithProfile(cfg, p)
	}

	// State dir override
	if o.StateDir != "" {
		cfg.StateDir = o.StateDir
	}
	if cfg.StateDir != "" {
		cfg.StateDir = filepath.Clean(cfg.StateDir)
	}

	// CLI overrides
	applyCLIOverrides(cfg, o.CLI)

	return cfg, nil
}

func applyCLIOverrides(cfg *Config, o CLIOverrides) {
	if o.DiscordWebhookURL != "" {
		cfg.Discord.WebhookURL = o.DiscordWebhookURL
	}
	if o.SlackWebhookURL != "" {
		cfg.Slack.WebhookURL = o.SlackWebhookURL
	}
	if o.SlackBotToken != "" {
		cfg.Slack.BotToken = o.SlackBotToken
	}
	if o.SlackChannel != "" {
		cfg.Slack.Channel = o.SlackChannel
	}
	if o.TelegramBotToken != "" {
		cfg.Telegram.BotToken = o.TelegramBotToken
	}
	if o.TelegramChatID != "" {
		cfg.Telegram.ChatID = o.TelegramChatID
	}
	if o.WebhookURL != "" {
		cfg.Webhook.URL = o.WebhookURL
	}
	if len(o.WebhookHeaders) > 0 {
		if cfg.Webhook.Headers == nil {
			cfg.Webhook.Headers = map[string]string{}
		}
		for _, kv := range o.WebhookHeaders {
			k, v, ok := splitKV(kv)
			if ok {
				cfg.Webhook.Headers[k] = v
			}
		}
	}
}

func splitKV(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

func mergeConfig(a Config, b Config) Config {
	// shallow merge, with some care for nested defaults
	if b.StateDir != "" {
		a.StateDir = b.StateDir
	}
	a.Transport = mergeTransport(a.Transport, b.Transport)
	a.Notify = mergeNotify(a.Notify, b.Notify)

	if b.Discord.WebhookURL != "" {
		a.Discord = b.Discord
	}
	if b.Slack.WebhookURL != "" || b.Slack.BotToken != "" || b.Slack.Channel != "" {
		a.Slack = mergeSlack(a.Slack, b.Slack)
	}
	if b.Telegram.BotToken != "" || b.Telegram.ChatID != "" || b.Telegram.ParseMode != "" {
		a.Telegram = mergeTelegram(a.Telegram, b.Telegram)
	}
	if b.Webhook.URL != "" || len(b.Webhook.Headers) > 0 {
		a.Webhook = mergeWebhook(a.Webhook, b.Webhook)
	}
	if b.EventOutput != "" {
		a.EventOutput = b.EventOutput
	}

	// profiles: replace if present
	if b.Profiles != nil {
		a.Profiles = b.Profiles
	}

	return a
}

func mergeTransport(a, b TransportConfig) TransportConfig {
	if b.Proxy != "" {
		a.Proxy = b.Proxy
	}
	if b.ProxyAuth != "" {
		a.ProxyAuth = b.ProxyAuth
	}
	if b.NoProxyEnv {
		a.NoProxyEnv = true
	}
	if b.InsecureTLS {
		a.InsecureTLS = true
	}
	return a
}

func mergeSlack(a, b SlackConfig) SlackConfig {
	if b.WebhookURL != "" {
		a.WebhookURL = b.WebhookURL
	}
	if b.BotToken != "" {
		a.BotToken = b.BotToken
	}
	if b.Channel != "" {
		a.Channel = b.Channel
	}
	return a
}

func mergeTelegram(a, b TelegramConfig) TelegramConfig {
	if b.BotToken != "" {
		a.BotToken = b.BotToken
	}
	if b.ChatID != "" {
		a.ChatID = b.ChatID
	}
	if b.ParseMode != "" {
		a.ParseMode = b.ParseMode
	}
	return a
}

func mergeWebhook(a, b WebhookConfig) WebhookConfig {
	if b.URL != "" {
		a.URL = b.URL
	}
	if b.Headers != nil {
		if a.Headers == nil {
			a.Headers = map[string]string{}
		}
		for k, v := range b.Headers {
			a.Headers[k] = v
		}
	}
	return a
}

func mergeNotify(a, b NotifyConfig) NotifyConfig {
	if len(b.Callbacks) > 0 {
		a.Callbacks = b.Callbacks
	}
	if b.NotifyEach != "" {
		a.NotifyEach = b.NotifyEach
	}
	if len(b.NotifyOn) > 0 {
		a.NotifyOn = b.NotifyOn
	}
	if b.Mode != "" {
		a.Mode = b.Mode
	}
	if b.StripANSI != "" {
		a.StripANSI = b.StripANSI
	}
	if b.StripProgress != "" {
		a.StripProgress = b.StripProgress
	}

	// Text
	if b.Text.Select != "" {
		a.Text.Select = b.Text.Select
	}
	if b.Text.HeadLines != 0 {
		a.Text.HeadLines = b.Text.HeadLines
	}
	if b.Text.TailLines != 0 {
		a.Text.TailLines = b.Text.TailLines
	}

	// Attach
	if b.Attach.Enabled != a.Attach.Enabled {
		a.Attach.Enabled = b.Attach.Enabled
	}
	if b.Attach.SplitMode != "" {
		a.Attach.SplitMode = b.Attach.SplitMode
	}
	if b.Attach.TailLines != 0 {
		a.Attach.TailLines = b.Attach.TailLines
	}
	if b.Attach.PartMaxBytes != nil {
		if a.Attach.PartMaxBytes == nil {
			a.Attach.PartMaxBytes = map[string]int{}
		}
		for k, v := range b.Attach.PartMaxBytes {
			a.Attach.PartMaxBytes[k] = v
		}
	}

	// Filters
	if len(b.Filters.Include) > 0 {
		a.Filters.Include = b.Filters.Include
	}
	if len(b.Filters.Exclude) > 0 {
		a.Filters.Exclude = b.Filters.Exclude
	}

	// Redaction
	a.Redaction = mergeRedaction(a.Redaction, b.Redaction)
	// Alerts
	a.Alerts = mergeAlerts(a.Alerts, b.Alerts)

	return a
}

func mergeRedaction(a, b RedactionConfig) RedactionConfig {
	if b.Defaults != a.Defaults {
		a.Defaults = b.Defaults
	}
	if len(b.Patterns) > 0 {
		a.Patterns = append(a.Patterns, b.Patterns...)
	}
	if b.File != "" {
		a.File = b.File
	}
	return a
}

func mergeAlerts(a, b AlertsConfig) AlertsConfig {
	if len(b.Patterns) > 0 {
		a.Patterns = b.Patterns
	}
	if b.IncludeContextLines != 0 {
		a.IncludeContextLines = b.IncludeContextLines
	}
	return a
}

func (c *Config) Clone() *Config {
	out := *c
	// deep-ish copies where needed
	if c.Notify.Callbacks != nil {
		out.Notify.Callbacks = append([]string{}, c.Notify.Callbacks...)
	}
	out.Notify.Filters.Include = append([]string{}, c.Notify.Filters.Include...)
	out.Notify.Filters.Exclude = append([]string{}, c.Notify.Filters.Exclude...)
	out.Notify.Redaction.Patterns = append([]string{}, c.Notify.Redaction.Patterns...)
	out.Notify.Alerts.Patterns = append([]string{}, c.Notify.Alerts.Patterns...)
	if c.Webhook.Headers != nil {
		out.Webhook.Headers = map[string]string{}
		for k, v := range c.Webhook.Headers {
			out.Webhook.Headers[k] = v
		}
	}
	if c.Notify.Attach.PartMaxBytes != nil {
		out.Notify.Attach.PartMaxBytes = map[string]int{}
		for k, v := range c.Notify.Attach.PartMaxBytes {
			out.Notify.Attach.PartMaxBytes[k] = v
		}
	}
	if c.Profiles != nil {
		out.Profiles = c.Profiles
	}
	return &out
}

func (c *Config) Validate() error {
	// Lightweight checks; deeper checks happen at client construction.
	if c.Notify.Mode == "" {
		return errors.New("notify.mode cannot be empty")
	}
	return nil
}

func HasAnyCallback(cb []string) bool {
	return len(cb) > 0
}
