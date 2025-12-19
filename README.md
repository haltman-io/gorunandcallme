# gorunandcallme v1.0.0-alpha

This current version is full of bugs that need fixing. Recommendation: **do not use**.

Feel free to contribute.

--

Notes:

- Does not work correctly on Windows or emulated environments.

- The background and jobs module is completely broken.

## Full help:

```
$ ./gorunandcallme

  /$$$$$$  /$$$$$$$   /$$$$$$   /$$$$$$  /$$      /$$
 /$$__  $$| $$__  $$ /$$__  $$ /$$__  $$| $$$    /$$$
| $$  \__/| $$  \ $$| $$  \ $$| $$  \__/| $$$$  /$$$$
| $$ /$$$$| $$$$$$$/| $$$$$$$$| $$      | $$ $$/$$ $$
| $$|_  $$| $$__  $$| $$__  $$| $$      | $$  $$$| $$
| $$  \ $$| $$  \ $$| $$  | $$| $$    $$| $$\  $ | $$
|  $$$$$$/| $$  | $$| $$  | $$|  $$$$$$/| $$ \/  | $$
 \______/ |__/  |__/|__/  |__/ \______/ |__/     |__/


haltman.io (https://github.com/haltman-io)

[codename: gorunandcallme] - [release: v1.0.0-stable]

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

Usage:
  gorunandcallme [flags] -- <command> [args...]
  gorunandcallme [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  job         Manage background jobs

Flags:
      --alert-context-lines int      On alert, include last N context lines. (default 25)
      --alert-on stringArray         Send immediate alert when line matches regex (repeatable).
      --attach                       Allow sending output as file attachment when needed. (default true)
      --attach-max-bytes string      Override per-platform max attachment bytes: discord=...,telegram=...,webhook=...
      --attach-split-mode string     When attachment exceeds platform limit: split|tail (default "split")
      --attach-tail-lines int        When attach-split-mode=tail: send only last N lines as a file. (default 5000)
      --background                   Run as a background job and detach from terminal.
      --callback strings             Callbacks to enable (comma-separated): discord,slack,telegram,webhook,all
      --command string               Command string (used by shell modes). Example: "ls -lah" or "cat a | b"
      --config string                Path to YAML config file.
      --cwd string                   Working directory for the child process.
      --debug                        Debug diagnostics to stderr (cannot be used with notifications).
      --discord-webhook-url string   Discord webhook URL (overrides config).
      --env stringArray              Extra env var for child process (KEY=VALUE). Repeatable.
      --event-output string          Write events to a JSONL file (line events, start/finish, notifications).
      --exec-mode string             Execution mode: direct|shell|bash|zsh|pwsh|cmd|custom (default "direct")
  -h, --help                         help for gorunandcallme
  -k, --insecure                     Disable TLS verification for notification requests (curl-style).
      --no-color                     Disable colors (tool UI + strips ANSI from child output).
      --no-proxy                     Ignore HTTP(S)_PROXY env vars for notification clients.
      --no-tty-output                Do not mirror child output to your terminal (useful for pure notification jobs).
      --notify-each string           Notify interval (supports: s,m,h,d,w,mo,y). Example: 10s, 5m, 1h, 1d, 1w.
      --notify-exclude stringArray   Exclude lines matching regex from notifications (repeatable).
      --notify-head-lines int        If head selection: send first N lines. (default 200)
      --notify-include stringArray   Only notify lines matching regex (repeatable).
      --notify-mode string           Notify mode: text-only|attach-only|auto|summary (default "auto")
      --notify-on strings            Lifecycle notifications: start,finish (repeatable or comma-separated).
      --notify-tail-lines int        If tail selection: send last N lines. (default 200)
      --notify-text-select string    Text selection: all|head|tail (default "all")
  -o, --output string                Save processed output results to file (sorted + dedup by default).
      --output-mode string           Output mode: sort-dedup|raw (default "sort-dedup")
      --profile string               Config profile name (from YAML). (default "default")
      --proxy string                 Proxy for notification requests (http://, https://, socks5://).
      --proxy-auth string            Proxy auth user:pass (for HTTP CONNECT or SOCKS5 auth).
      --redact stringArray           Add redaction regex pattern (repeatable).
      --redact-defaults              Enable default redaction patterns. (default true)
      --redact-file string           Load redaction patterns from file (one regex per line).
      --shell string                 Custom shell path when --exec-mode=custom (e.g., /bin/bash, /usr/bin/zsh).
  -s, --silent                       Disable banner and non-essential UI output.
      --slack-bot-token string       Slack bot token for file uploads (optional).
      --slack-channel string         Slack channel ID/name for file uploads (optional).
      --slack-webhook-url string     Slack incoming webhook URL (overrides config).
      --state-dir string             State directory for jobs, logs, offsets (default: ~/.gorunandcallme).
      --stdin string                 Stdin mode: auto|on|off (pipeline-aware). (default "auto")
      --strip-ansi string            Strip ANSI escape codes: auto|always|never. (default "auto")
      --strip-progress string        Strip progress/spinner noise (carriage returns): auto|always|never. (default "auto")
      --telegram-bot-token string    Telegram bot token (overrides config).
      --telegram-chat-id string      Telegram chat ID (overrides config).
      --threads int                  Number of commands to run in parallel (scheduler). Default: 1. (default 1)
  -v, --verbose                      Verbose diagnostics to stderr (cannot be used with notifications).
      --webhook-header stringArray   Generic webhook header KEY=VALUE (repeatable).
      --webhook-url string           Generic webhook URL (overrides config).

Use "gorunandcallme [command] --help" for more information about a command.
```

## Examples

```
$ ./gorunandcallme --exec-mode direct -- echo "hello"

      ::::::::  :::::::::      :::      ::::::::    :::   :::
    :+:    :+: :+:    :+:   :+: :+:   :+:    :+:  :+:+: :+:+:
   +:+        +:+    +:+  +:+   +:+  +:+        +:+ +:+:+ +:+
  :#:        +#++:++#:  +#++:++#++: +#+        +#+  +:+  +#+
 +#+   +#+# +#+    +#+ +#+     +#+ +#+        +#+       +#+
#+#    #+# #+#    #+# #+#     #+# #+#    #+# #+#       #+#
########  ###    ### ###     ###  ########  ###       ###


haltman.io (https://github.com/haltman-io)

[codename: gorunandcallme] - [release: v1.0.0-stable]

hello
```

```
$ ./gorunandcallme --exec-mode direct -- printf "a\nb\nc\n"

 ██████╗ ██████╗  █████╗  ██████╗███╗   ███╗
██╔════╝ ██╔══██╗██╔══██╗██╔════╝████╗ ████║
██║  ███╗██████╔╝███████║██║     ██╔████╔██║
██║   ██║██╔══██╗██╔══██║██║     ██║╚██╔╝██║
╚██████╔╝██║  ██║██║  ██║╚██████╗██║ ╚═╝ ██║
 ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝     ╚═╝


haltman.io (https://github.com/haltman-io)

[codename: gorunandcallme] - [release: v1.0.0-stable]

a
b
c
```

```
$ ./gorunandcallme --exec-mode direct --env FOO=bar -- sh -lc 'echo "$FOO"'

 ______     ______     ______     ______     __    __
/\  ___\   /\  == \   /\  __ \   /\  ___\   /\ "-./  \
\ \ \__ \  \ \  __<   \ \  __ \  \ \ \____  \ \ \-./\ \
 \ \_____\  \ \_\ \_\  \ \_\ \_\  \ \_____\  \ \_\ \ \_\
  \/_____/   \/_/ /_/   \/_/\/_/   \/_____/   \/_/  \/_/


haltman.io (https://github.com/haltman-io)

[codename: gorunandcallme] - [release: v1.0.0-stable]

bar
```

```
$ ./gorunandcallme --exec-mode shell --command 'printf "a\nb\nc\n" | wc -l'

  ________  ___  _______  ___
 / ___/ _ \/ _ |/ ___/  |/  /
/ (_ / , _/ __ / /__/ /|_/ /
\___/_/|_/_/ |_\___/_/  /_/


haltman.io (https://github.com/haltman-io)

[codename: gorunandcallme] - [release: v1.0.0-stable]

3
```

## Examples (receive notifications)
```
Send text batches (small output)
Command:
  ./gorunandcallme --callback webhook --webhook-url "<WEBHOOK_URL>" --notify-each 2s --notify-mode text-only \
    --exec-mode shell --command 'for i in $(seq 1 5); do echo "line $i"; sleep 0.3; done'
Expected:
  - You receive webhook messages every ~2 seconds (batched output).
  - Message body includes code block formatting (``` ... ```).
  - No ANSI garbage.
```

```
Lifecycle notifications start/finish
Command:
  ./gorunandcallme --callback webhook --webhook-url "<WEBHOOK_URL>" --notify-on start,finish \
    --notify-each 2s --exec-mode shell --command 'echo "hello"; sleep 1; echo "bye"'
Expected:
  - A "started" notification with the full command line (gorunandcallme invocation).
  - A "finished" notification including exit code + plan details.
```

```
Discord (webhook) - text-only
Command:
  ./gorunandcallme --callback discord --discord-webhook-url "<DISCORD_WEBHOOK_URL>" --notify-each 2s --notify-mode text-only \
    --exec-mode shell --command 'for i in $(seq 1 10); do echo "d-$i"; sleep 0.2; done'
Expected:
  - Discord receives batched code block messages.
  - No missing messages, no rate limit crashes.
```

```
Slack (incoming webhook) - text-only
Command:
  ./gorunandcallme --callback slack --slack-webhook-url "<SLACK_WEBHOOK_URL>" --notify-each 2s --notify-mode text-only \
    --exec-mode shell --command 'for i in $(seq 1 10); do echo "s-$i"; sleep 0.2; done'
Expected:
  - Slack receives batched code block messages.
```

```
Telegram (bot) - text-only
Command:
  ./gorunandcallme --callback telegram --telegram-bot-token "<TELEGRAM_BOT_TOKEN>" --telegram-chat-id "<TELEGRAM_CHAT_ID>" \
    --notify-each 2s --notify-mode text-only \
    --exec-mode shell --command 'for i in $(seq 1 10); do echo "t-$i"; sleep 0.2; done'
Expected:
  - Telegram receives messages.
  - For long messages: Telegram may truncate at 4096; verify tool chunking/truncation behavior as designed.
```

```
Attach-only + split mode (large output)
Command:
  ./gorunandcallme --callback webhook --webhook-url "<WEBHOOK_URL>" --notify-mode attach-only --attach \
    --attach-split-mode split --notify-each 2s \
    --exec-mode shell --command 'python3 - << "PY"
for i in range(20000):
  print("line", i)
PY'
Expected:
  - Tool sends multiple attachments (part 001/XXX, part 002/XXX, etc.) until complete.
  - Verify it does NOT exceed platform per-file limit. If it does, split must happen.
```

```
Broadcast to multiple platforms
Command:
  ./gorunandcallme --callback "discord,telegram" \
    --discord-webhook-url "<DISCORD_WEBHOOK_URL>" \
    --telegram-bot-token "<TELEGRAM_BOT_TOKEN>" --telegram-chat-id "<TELEGRAM_CHAT_ID>" \
    --notify-each 2s --exec-mode shell --command 'for i in $(seq 1 10); do echo "multi-$i"; sleep 0.2; done'
Expected:
  - Same batched output arrives to BOTH destinations.
```

```
HTTP proxy without auth (notifications)
Command:
  ./gorunandcallme --callback webhook --webhook-url "<WEBHOOK_URL>" --proxy "<PROXY_URL>" --notify-each 2s \
    --exec-mode shell --command 'echo "proxy-test"'
Expected:
  - Your proxy (e.g., Burp) sees outbound notify HTTP requests.
  - Notification still arrives.

```

```
TLS insecure (curl-style)
Command:
  ./gorunandcallme --callback webhook --webhook-url "<WEBHOOK_URL>" --insecure \
    --exec-mode shell --command 'echo "insecure-tls-test"'
Expected:
  - If your webhook uses self-signed/invalid cert, this should still work.
  - If webhook has valid cert, should still work.
```

```
NO-TTY OUTPUT (notification-only job)
-----------------------------------------
Command:
  ./gorunandcallme --no-tty-output --callback webhook --webhook-url "<WEBHOOK_URL>" --notify-each 2s \
    --exec-mode shell --command 'for i in $(seq 1 5); do echo "ntty-$i"; sleep 0.5; done'
Expected:
  - Terminal stays quiet (or minimal).
  - Notifications still arrive with the output.
```

```

```

```

```

```

```

```

```

