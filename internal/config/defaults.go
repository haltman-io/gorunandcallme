package config

import "github.com/haltman-io/gorunandcallme/internal/util"

func DefaultConfig() *Config {
	return &Config{
		StateDir: "",
		Transport: TransportConfig{
			Proxy:      "",
			ProxyAuth:  "",
			NoProxyEnv: false,
			InsecureTLS: false,
		},
		Notify: NotifyConfig{
			Callbacks:     nil,
			NotifyEach:    "",
			NotifyOn:      []string{"start", "finish"},
			Mode:          "auto",
			StripANSI:     "auto",
			StripProgress: "auto",
			Text: NotifyTextConfig{
				Select:    "all",
				HeadLines: 200,
				TailLines: 200,
			},
			Attach: NotifyAttachConfig{
				Enabled:   true,
				SplitMode: "split",
				TailLines: 5000,
				PartMaxBytes: map[string]int{
					"discord":  8000000,
					"telegram": 45000000,
					"webhook":  10000000,
				},
			},
			Filters: NotifyFilterConfig{
				Include: nil,
				Exclude: nil,
			},
			Redaction: RedactionConfig{
				Defaults: true,
				Patterns: nil,
				File:     "",
			},
			Alerts: AlertsConfig{
				Patterns:            nil,
				IncludeContextLines: 25,
			},
		},
		Discord: DiscordConfig{},
		Slack:   SlackConfig{},
		Telegram: TelegramConfig{
			ParseMode: "MarkdownV2",
		},
		Webhook: WebhookConfig{
			Headers: map[string]string{},
		},
		EventOutput: "",
		Profiles:    map[string]*ProfileConfig{},
	}
}

func MergeBaseWithProfile(base *Config, p *ProfileConfig) *Config {
	out := base.Clone()
	if p == nil {
		return out
	}

	if p.Notify != nil {
		out.Notify = util.Merge(out.Notify, *p.Notify)
	}
	if p.Discord != nil {
		out.Discord = util.Merge(out.Discord, *p.Discord)
	}
	if p.Slack != nil {
		out.Slack = util.Merge(out.Slack, *p.Slack)
	}
	if p.Telegram != nil {
		out.Telegram = util.Merge(out.Telegram, *p.Telegram)
	}
	if p.Webhook != nil {
		out.Webhook = util.Merge(out.Webhook, *p.Webhook)
	}

	return out
}
