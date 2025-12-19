package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/haltman-io/gorunandcallme/internal/config"
	"github.com/haltman-io/gorunandcallme/internal/execx"
	"github.com/haltman-io/gorunandcallme/internal/job"
	"github.com/haltman-io/gorunandcallme/internal/notify"
	"github.com/haltman-io/gorunandcallme/internal/util"
	"github.com/spf13/cobra"
)

type RootOptions struct {
	Profile  string
	Config   string
	StateDir string

	Silent  bool
	NoColor bool
	Verbose bool
	Debug   bool

	Background bool

	// Execution
	ExecMode    string
	Shell       string
	CommandStr  string
	UseStdin    string // auto|on|off
	Parallel    int
	WorkingDir  string
	EnvPairs    []string
	StripANSI   string // auto|always|never (terminal and notify)
	StripProg   string // auto|always|never
	EventOutput string

	// Notification options
	Callbacks           []string
	NotifyEach          string
	NotifyOn            []string // start,finish
	NotifyMode          string   // text-only|attach-only|auto|summary
	NotifyTextSelect    string   // all|head|tail
	NotifyHeadLines     int
	NotifyTailLines     int
	AttachEnabled       bool
	AttachSplitMode     string // split|tail
	AttachTailLines     int
	AttachMaxBytes      string // "discord=8000000,telegram=45000000,webhook=10000000"
	NotifyIncludeRegex  []string
	NotifyExcludeRegex  []string
	RedactDefaults      bool
	RedactPatterns      []string
	RedactFile          string
	AlertPatterns       []string
	AlertContextLines   int
	OutputFile          string
	OutputMode          string // sort-dedup|raw
	NoTTYOutput         bool
	CallbackRoundRobin  bool // broadcast by default; kept for semantics
	DisableNotifyIfLogs bool // internal guard; always true
	Proxy               string
	ProxyAuth           string
	NoProxyEnv          bool
	InsecureTLS         bool

	// Inline platform config overrides
	DiscordWebhookURL string
	SlackWebhookURL   string
	SlackBotToken     string
	SlackChannel      string
	TelegramBotToken  string
	TelegramChatID    string
	WebhookURL        string
	WebhookHeaders    []string
}

func Main() int {
	root := &RootOptions{
		Parallel:          1,
		UseStdin:          "auto",
		ExecMode:          "direct",
		StripANSI:         "auto",
		StripProg:         "auto",
		NotifyMode:        "auto",
		NotifyTextSelect:  "all",
		NotifyHeadLines:   200,
		NotifyTailLines:   200,
		AttachEnabled:     true,
		AttachSplitMode:   "split",
		AttachTailLines:   5000,
		RedactDefaults:    true,
		AlertContextLines: 25,
		OutputMode:        "sort-dedup",
	}

	cmd := buildRootCmd(root)
	if err := cmd.Execute(); err != nil {
		// Cobra already prints usage on some errors; keep exit code stable.
		return 1
	}
	return 0
}

func buildRootCmd(o *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gorunandcallme [flags] -- <command> [args...]",
		Short: "Run external tools, monitor output, and notify to Discord/Slack/Telegram/Webhooks on schedule.",
		Long: strings.TrimSpace(`
gorunandcallme is a CLI wrapper designed to run external commands (security tools, automations, pipelines),
stream output in real-time, optionally strip ANSI/progress noise, and dispatch batched notifications to multiple
destinations with rate-limit aware sending, truncation, chunking, or file attachments.

Direct (strict) execution is recommended:
  gorunandcallme --exec-mode direct -- nuclei -u https://example.com -silent

Shell execution is supported when you need shell features:
  gorunandcallme --exec-mode shell --command "cat subdomains.txt | httpx -silent | nuclei -silent"

Background jobs:
  gorunandcallme --background --notify-each 30s --callback telegram -- nmap -sV example.com
  gorunandcallme job follow <job-id>
		`),
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          func(cmd *cobra.Command, args []string) error { return nil },
		RunE: func(cmd *cobra.Command, args []string) error {
			if missingCommand(o, args, cmd) {
				return cmd.Help()
			}

			ui := NewUI(o.NoColor, o.Verbose, o.Debug)
			ui.BannerIfAllowed(o.Silent)

			// Load config (optional) and merge with CLI.
			cfg, err := loadMergedConfig(o)
			if err != nil {
				return err
			}

			stateDir := resolveStateDir(o, cfg)

			// Validate conflicts: debug/verbose cannot be enabled with notifications.
			if (o.Debug || o.Verbose) && notify.HasCallbacks(o.Callbacks, cfg) {
				return errors.New("debug/verbose output cannot be used with notifications (callbacks) enabled. Disable --debug/--verbose or remove --callback/notify config")
			}

			// Background: spawn detached child and return job-id.
			if o.Background {
				st, err := job.NewStore(stateDir)
				if err != nil {
					return err
				}

				exe, err := os.Executable()
				if err != nil {
					return err
				}

				argsForChild := stripBackgroundFlag(os.Args[1:])

				res, err := job.SpawnBackground(cmd.Context(), job.SpawnOptions{
					RunnerPath: exe,
					RunnerArgs: argsForChild,
					Workdir:    o.WorkingDir,
					Store:      st,
				})
				if err != nil {
					return err
				}
				ui.Info("Started background job: %s", res.JobID)
				ui.Info("Follow: gorunandcallme job follow %s", res.JobID)
				return nil
			}

			// Foreground run
			return runForeground(ui, cfg, stateDir, o, args)
		},
	}

	// Global flags
	cmd.Flags().StringVar(&o.Profile, "profile", "default", "Config profile name (from YAML).")
	cmd.Flags().StringVar(&o.Config, "config", "", "Path to YAML config file.")
	cmd.PersistentFlags().StringVar(&o.StateDir, "state-dir", "", "State directory for jobs, logs, offsets (default: ~/.gorunandcallme).")

	cmd.Flags().BoolVarP(&o.Silent, "silent", "s", false, "Disable banner and non-essential UI output.")
	cmd.Flags().BoolVar(&o.NoColor, "no-color", false, "Disable colors (tool UI + strips ANSI from child output).")
	cmd.Flags().BoolVarP(&o.Verbose, "verbose", "v", false, "Verbose diagnostics to stderr (cannot be used with notifications).")
	cmd.Flags().BoolVar(&o.Debug, "debug", false, "Debug diagnostics to stderr (cannot be used with notifications).")

	cmd.Flags().BoolVar(&o.Background, "background", false, "Run as a background job and detach from terminal.")

	// Execution flags
	cmd.Flags().StringVar(&o.ExecMode, "exec-mode", o.ExecMode, "Execution mode: direct|shell|bash|zsh|pwsh|cmd|custom")
	cmd.Flags().StringVar(&o.Shell, "shell", "", "Custom shell path when --exec-mode=custom (e.g., /bin/bash, /usr/bin/zsh).")
	cmd.Flags().StringVar(&o.CommandStr, "command", "", "Command string (used by shell modes). Example: \"ls -lah\" or \"cat a | b\"")
	cmd.Flags().StringVar(&o.UseStdin, "stdin", o.UseStdin, "Stdin mode: auto|on|off (pipeline-aware).")
	cmd.Flags().IntVar(&o.Parallel, "threads", o.Parallel, "Number of commands to run in parallel (scheduler). Default: 1.")
	cmd.Flags().StringVar(&o.WorkingDir, "cwd", "", "Working directory for the child process.")
	cmd.Flags().StringArrayVar(&o.EnvPairs, "env", nil, "Extra env var for child process (KEY=VALUE). Repeatable.")
	cmd.Flags().StringVar(&o.StripANSI, "strip-ansi", o.StripANSI, "Strip ANSI escape codes: auto|always|never.")
	cmd.Flags().StringVar(&o.StripProg, "strip-progress", o.StripProg, "Strip progress/spinner noise (carriage returns): auto|always|never.")
	cmd.Flags().StringVar(&o.EventOutput, "event-output", "", "Write events to a JSONL file (line events, start/finish, notifications).")
	cmd.Flags().BoolVar(&o.NoTTYOutput, "no-tty-output", false, "Do not mirror child output to your terminal (useful for pure notification jobs).")

	// Notifications + platform flags
	cmd.Flags().StringSliceVar(&o.Callbacks, "callback", nil, "Callbacks to enable (comma-separated): discord,slack,telegram,webhook,all")
	cmd.Flags().StringVar(&o.NotifyEach, "notify-each", "", "Notify interval (supports: s,m,h,d,w,mo,y). Example: 10s, 5m, 1h, 1d, 1w.")
	cmd.Flags().StringSliceVar(&o.NotifyOn, "notify-on", nil, "Lifecycle notifications: start,finish (repeatable or comma-separated).")
	cmd.Flags().StringVar(&o.NotifyMode, "notify-mode", o.NotifyMode, "Notify mode: text-only|attach-only|auto|summary")
	cmd.Flags().StringVar(&o.NotifyTextSelect, "notify-text-select", o.NotifyTextSelect, "Text selection: all|head|tail")
	cmd.Flags().IntVar(&o.NotifyHeadLines, "notify-head-lines", o.NotifyHeadLines, "If head selection: send first N lines.")
	cmd.Flags().IntVar(&o.NotifyTailLines, "notify-tail-lines", o.NotifyTailLines, "If tail selection: send last N lines.")

	cmd.Flags().BoolVar(&o.AttachEnabled, "attach", o.AttachEnabled, "Allow sending output as file attachment when needed.")
	cmd.Flags().StringVar(&o.AttachSplitMode, "attach-split-mode", o.AttachSplitMode, "When attachment exceeds platform limit: split|tail")
	cmd.Flags().IntVar(&o.AttachTailLines, "attach-tail-lines", o.AttachTailLines, "When attach-split-mode=tail: send only last N lines as a file.")
	cmd.Flags().StringVar(&o.AttachMaxBytes, "attach-max-bytes", "", "Override per-platform max attachment bytes: discord=...,telegram=...,webhook=...")

	cmd.Flags().StringArrayVar(&o.NotifyIncludeRegex, "notify-include", nil, "Only notify lines matching regex (repeatable).")
	cmd.Flags().StringArrayVar(&o.NotifyExcludeRegex, "notify-exclude", nil, "Exclude lines matching regex from notifications (repeatable).")

	cmd.Flags().BoolVar(&o.RedactDefaults, "redact-defaults", o.RedactDefaults, "Enable default redaction patterns.")
	cmd.Flags().StringArrayVar(&o.RedactPatterns, "redact", nil, "Add redaction regex pattern (repeatable).")
	cmd.Flags().StringVar(&o.RedactFile, "redact-file", "", "Load redaction patterns from file (one regex per line).")

	cmd.Flags().StringArrayVar(&o.AlertPatterns, "alert-on", nil, "Send immediate alert when line matches regex (repeatable).")
	cmd.Flags().IntVar(&o.AlertContextLines, "alert-context-lines", o.AlertContextLines, "On alert, include last N context lines.")

	cmd.Flags().StringVarP(&o.OutputFile, "output", "o", "", "Save processed output results to file (sorted + dedup by default).")
	cmd.Flags().StringVar(&o.OutputMode, "output-mode", o.OutputMode, "Output mode: sort-dedup|raw")

	// Network options for notification HTTP clients only
	cmd.Flags().StringVar(&o.Proxy, "proxy", "", "Proxy for notification requests (http://, https://, socks5://).")
	cmd.Flags().StringVar(&o.ProxyAuth, "proxy-auth", "", "Proxy auth user:pass (for HTTP CONNECT or SOCKS5 auth).")
	cmd.Flags().BoolVar(&o.NoProxyEnv, "no-proxy", false, "Ignore HTTP(S)_PROXY env vars for notification clients.")
	cmd.Flags().BoolVarP(&o.InsecureTLS, "insecure", "k", false, "Disable TLS verification for notification requests (curl-style).")

	// Inline platform config
	cmd.Flags().StringVar(&o.DiscordWebhookURL, "discord-webhook-url", "", "Discord webhook URL (overrides config).")
	cmd.Flags().StringVar(&o.SlackWebhookURL, "slack-webhook-url", "", "Slack incoming webhook URL (overrides config).")
	cmd.Flags().StringVar(&o.SlackBotToken, "slack-bot-token", "", "Slack bot token for file uploads (optional).")
	cmd.Flags().StringVar(&o.SlackChannel, "slack-channel", "", "Slack channel ID/name for file uploads (optional).")
	cmd.Flags().StringVar(&o.TelegramBotToken, "telegram-bot-token", "", "Telegram bot token (overrides config).")
	cmd.Flags().StringVar(&o.TelegramChatID, "telegram-chat-id", "", "Telegram chat ID (overrides config).")
	cmd.Flags().StringVar(&o.WebhookURL, "webhook-url", "", "Generic webhook URL (overrides config).")
	cmd.Flags().StringArrayVar(&o.WebhookHeaders, "webhook-header", nil, "Generic webhook header KEY=VALUE (repeatable).")

	// Job subcommands
	cmd.AddCommand(buildJobCmd(o))

	installHelpWithBanner(cmd, o)

	return cmd
}

func buildJobCmd(o *RootOptions) *cobra.Command {
	jobCmd := &cobra.Command{
		Use:   "job",
		Short: "Manage background jobs",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List background jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := newJobStore(o)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			return job.CmdList(cmd.OutOrStdout(), st, job.ListOptions{Limit: limit})
		},
	}
	listCmd.Flags().Int("limit", 0, "Show at most N jobs (0 = all)")
	jobCmd.AddCommand(listCmd)

	statusCmd := &cobra.Command{
		Use:   "status <job-id>",
		Short: "Show job status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := newJobStore(o)
			if err != nil {
				return err
			}
			return job.CmdStatus(cmd.OutOrStdout(), st, args[0])
		},
	}
	jobCmd.AddCommand(statusCmd)

	followCmd := &cobra.Command{
		Use:   "follow <job-id>",
		Short: "Follow job log output",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := newJobStore(o)
			if err != nil {
				return err
			}
			tail, _ := cmd.Flags().GetInt("tail")
			poll, _ := cmd.Flags().GetDuration("poll")
			once, _ := cmd.Flags().GetBool("once")

			return job.CmdFollow(args[0], job.FollowOptions{
				Follow:   !once,
				Poll:     poll,
				Tail:     tail,
				Stdout:   cmd.OutOrStdout(),
				Ctx:      cmd.Context(),
				JobStore: st,
			})
		},
	}
	followCmd.Flags().Int("tail", 20, "Tail last N lines before following (0 = disable)")
	followCmd.Flags().Duration("poll", 500*time.Millisecond, "Polling interval while following")
	followCmd.Flags().Bool("once", false, "Print log once without following")
	jobCmd.AddCommand(followCmd)

	stopCmd := &cobra.Command{
		Use:   "stop <job-id>",
		Short: "Stop a background job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := newJobStore(o)
			if err != nil {
				return err
			}
			return job.CmdStop(cmd.OutOrStdout(), st, args[0])
		},
	}
	jobCmd.AddCommand(stopCmd)

	purgeCmd := &cobra.Command{
		Use:   "purge <job-id>",
		Short: "Delete a job and its logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := newJobStore(o)
			if err != nil {
				return err
			}
			return job.CmdPurge(cmd.OutOrStdout(), st, args[0])
		},
	}
	jobCmd.AddCommand(purgeCmd)

	return jobCmd
}

func runForeground(ui *UI, cfg *config.Config, stateDir string, o *RootOptions, args []string) error {
	// Merge CLI notify overrides over config notify.
	runtimeCfg := cfg.Clone()

	// Callbacks from CLI override config if set.
	if len(o.Callbacks) > 0 {
		runtimeCfg.Notify.Callbacks = util.NormalizeCSV(o.Callbacks)
	}

	// Apply notify interval override
	if o.NotifyEach != "" {
		runtimeCfg.Notify.NotifyEach = o.NotifyEach
	}
	if len(o.NotifyOn) > 0 {
		runtimeCfg.Notify.NotifyOn = util.NormalizeCSV(o.NotifyOn)
	}
	if o.NotifyMode != "" {
		runtimeCfg.Notify.Mode = o.NotifyMode
	}

	runtimeCfg.Notify.Text.Select = o.NotifyTextSelect
	runtimeCfg.Notify.Text.HeadLines = o.NotifyHeadLines
	runtimeCfg.Notify.Text.TailLines = o.NotifyTailLines

	runtimeCfg.Notify.Attach.Enabled = o.AttachEnabled
	runtimeCfg.Notify.Attach.SplitMode = o.AttachSplitMode
	runtimeCfg.Notify.Attach.TailLines = o.AttachTailLines
	if o.AttachMaxBytes != "" {
		runtimeCfg.Notify.Attach.PartMaxBytes = util.ParseKVIntMap(o.AttachMaxBytes)
	}

	if len(o.NotifyIncludeRegex) > 0 {
		runtimeCfg.Notify.Filters.Include = o.NotifyIncludeRegex
	}
	if len(o.NotifyExcludeRegex) > 0 {
		runtimeCfg.Notify.Filters.Exclude = o.NotifyExcludeRegex
	}

	runtimeCfg.Notify.Redaction.Defaults = o.RedactDefaults
	if len(o.RedactPatterns) > 0 {
		runtimeCfg.Notify.Redaction.Patterns = append(runtimeCfg.Notify.Redaction.Patterns, o.RedactPatterns...)
	}
	runtimeCfg.Notify.Redaction.File = o.RedactFile

	if len(o.AlertPatterns) > 0 {
		runtimeCfg.Notify.Alerts.Patterns = o.AlertPatterns
	}
	runtimeCfg.Notify.Alerts.IncludeContextLines = o.AlertContextLines

	runtimeCfg.Notify.StripANSI = o.StripANSI
	runtimeCfg.Notify.StripProgress = o.StripProg

	// Transport options for notifications
	runtimeCfg.Transport.Proxy = o.Proxy
	runtimeCfg.Transport.ProxyAuth = o.ProxyAuth
	runtimeCfg.Transport.NoProxyEnv = o.NoProxyEnv
	runtimeCfg.Transport.InsecureTLS = o.InsecureTLS

	// Determine command plan
	plan, err := execx.BuildPlan(execx.PlanOptions{
		ExecMode:   o.ExecMode,
		Shell:      o.Shell,
		CommandStr: o.CommandStr,
		Args:       args,
		CWD:        o.WorkingDir,
		EnvPairs:   o.EnvPairs,
		StdinMode:  o.UseStdin,
		NoColor:    o.NoColor,
	})
	if err != nil {
		return err
	}

	// Build dispatcher (optional)
	var disp *notify.Dispatcher
	var agg *notify.Aggregator
	var evt *notify.EventSink

	if notify.HasCallbacks(nil, runtimeCfg) {
		httpc, err := notify.NewHTTPClient(runtimeCfg.Transport)
		if err != nil {
			return err
		}

		clients, err := notify.BuildClients(httpc, runtimeCfg)
		if err != nil {
			return err
		}

		disp = notify.NewDispatcher(notify.DispatcherOptions{
			Clients: clients,
			UI:      ui,
		})

		red, err := notify.NewRedactor(runtimeCfg.Notify.Redaction)
		if err != nil {
			return err
		}
		filt, err := notify.NewFilters(runtimeCfg.Notify.Filters)
		if err != nil {
			return err
		}
		alerts, err := notify.NewAlerts(runtimeCfg.Notify.Alerts)
		if err != nil {
			return err
		}

		agg, err = notify.NewAggregator(notify.AggregatorOptions{
			Config:   runtimeCfg.Notify,
			Redactor: red,
			Filters:  filt,
			Alerts:   alerts,
			UI:       ui,
			Dispatch: disp,
		})
		if err != nil {
			return err
		}

		evt = notify.NewEventSink(runtimeCfg.EventOutput, ui)
	} else {
		evt = notify.NewEventSink(runtimeCfg.EventOutput, ui)
	}

	// Start/finish notifications (semantic)
	fullCmd := strings.Join(os.Args, " ")
	if agg != nil {
		if notify.WantsLifecycle(runtimeCfg.Notify.NotifyOn, "start") {
			agg.SendLifecycle("started", fullCmd, plan.Describe())
		}
	}

	// Run plan (scheduler)
	runner := execx.NewScheduler(execx.SchedulerOptions{
		Threads:    o.Parallel,
		UI:         ui,
		EventSink:  evt,
		NoTTY:      o.NoTTYOutput,
		StripANSI:  util.ShouldStripANSI(o.NoColor, o.StripANSI),
		StripProg:  util.ShouldStripProgress(o.StripProg),
		NotifyHook: agg,
		OutputFile: o.OutputFile,
		OutputMode: o.OutputMode,
	})

	exitCode, runErr := runner.Run(plan)
	if runErr != nil {
		ui.Error("%v", runErr)
	}

	if agg != nil {
		agg.FlushAll("final")
		if notify.WantsLifecycle(runtimeCfg.Notify.NotifyOn, "finish") {
			agg.SendLifecycle("finished", fullCmd, fmt.Sprintf("%s | exit=%d", plan.Describe(), exitCode))
		}
		disp.Close()
	}

	if evt != nil {
		evt.Close()
	}

	if exitCode != 0 {
		return fmt.Errorf("child exited with code %d", exitCode)
	}
	return nil
}

func loadMergedConfig(o *RootOptions) (*config.Config, error) {
	return config.LoadMerged(config.LoadOptions{
		ConfigPath: o.Config,
		Profile:    o.Profile,
		StateDir:   o.StateDir,
		CLI: config.CLIOverrides{
			DiscordWebhookURL: o.DiscordWebhookURL,
			SlackWebhookURL:   o.SlackWebhookURL,
			SlackBotToken:     o.SlackBotToken,
			SlackChannel:      o.SlackChannel,
			TelegramBotToken:  o.TelegramBotToken,
			TelegramChatID:    o.TelegramChatID,
			WebhookURL:        o.WebhookURL,
			WebhookHeaders:    o.WebhookHeaders,
		},
	})
}

func resolveStateDir(o *RootOptions, cfg *config.Config) string {
	stateDir := ""
	if cfg != nil {
		stateDir = cfg.StateDir
	}
	if o.StateDir != "" {
		stateDir = o.StateDir
	}
	if stateDir == "" {
		stateDir = filepath.Join(util.UserHomeDirOrDot(), ".gorunandcallme")
	}
	return stateDir
}

func newJobStore(o *RootOptions) (*job.Store, error) {
	cfg, err := loadMergedConfig(o)
	if err != nil {
		return nil, err
	}
	stateDir := resolveStateDir(o, cfg)
	return job.NewStore(stateDir)
}

func stripBackgroundFlag(args []string) []string {
	out := make([]string, 0, len(args))
	skipNext := false

	for i, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if a == "--" {
			out = append(out, args[i:]...)
			break
		}

		if a == "--background" {
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				skipNext = true
			}
			continue
		}
		if strings.HasPrefix(a, "--background=") {
			continue
		}

		out = append(out, a)
	}

	return out
}

func missingCommand(o *RootOptions, args []string, cmd *cobra.Command) bool {
	if cmd.Name() == "job" || cmd.Parent() != nil && cmd.Parent().Name() == "job" {
		return false
	}
	if o.CommandStr != "" || len(args) > 0 {
		return false
	}
	if cmd.Flags().Changed("from-file") {
		return false
	}
	return true
}

func installHelpWithBanner(cmd *cobra.Command, o *RootOptions) {
	defaultHelp := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		ui := NewUI(o.NoColor, o.Verbose, o.Debug)
		ui.BannerIfAllowed(o.Silent)
		defaultHelp(c, args)
	})
}
