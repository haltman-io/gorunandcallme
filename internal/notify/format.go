package notify

import (
	"bytes"
	"fmt"
	"strings"
)

type TextSelectMode string

const (
	SelectAll  TextSelectMode = "all"
	SelectHead TextSelectMode = "head"
	SelectTail TextSelectMode = "tail"
)

func WrapCodeBlockMarkdown(text string) string {
	// Telegram MarkdownV2: backticks are allowed but must be escaped inside content.
	// We escape per-platform in telegram client; here we keep raw.
	return "```\n" + text + "\n```"
}

func HeadLines(lines []string, n int) []string {
	if n <= 0 || n >= len(lines) {
		return lines
	}
	return lines[:n]
}

func TailLines(lines []string, n int) []string {
	if n <= 0 || n >= len(lines) {
		return lines
	}
	return lines[len(lines)-n:]
}

func JoinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

func ChunkByChars(s string, max int) []string {
	if max <= 0 || len(s) <= max {
		return []string{s}
	}
	out := []string{}
	for len(s) > 0 {
		if len(s) <= max {
			out = append(out, s)
			break
		}
		out = append(out, s[:max])
		s = s[max:]
	}
	return out
}

func BuildAttachmentParts(lines []string, maxBytes int) [][]byte {
	// Split by bytes, keeping line boundaries where possible.
	if maxBytes <= 0 {
		// fallback: single file
		return [][]byte{[]byte(JoinLines(lines) + "\n")}
	}
	var parts [][]byte
	var buf bytes.Buffer
	for _, l := range lines {
		line := []byte(l + "\n")
		if buf.Len()+len(line) > maxBytes && buf.Len() > 0 {
			parts = append(parts, append([]byte{}, buf.Bytes()...))
			buf.Reset()
		}
		// If a single line exceeds maxBytes, force it as its own part (may exceed, but unavoidable).
		if len(line) > maxBytes && buf.Len() == 0 {
			parts = append(parts, line)
			continue
		}
		_, _ = buf.Write(line)
	}
	if buf.Len() > 0 {
		parts = append(parts, append([]byte{}, buf.Bytes()...))
	}
	return parts
}

func Summary(lines []string, top int) string {
	// Simple summary: show counts + top lines.
	if top <= 0 {
		top = 30
	}
	n := len(lines)
	if n == 0 {
		return "No output captured."
	}
	show := lines
	if len(show) > top {
		show = show[:top]
	}
	return fmt.Sprintf("Lines: %d\n\nTop %d:\n%s", n, len(show), JoinLines(show))
}
