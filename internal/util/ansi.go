package util

import (
	"regexp"
	"strings"
)

var (
	// CSI: ESC [ ... command
	ansiCSI = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	// OSC: ESC ] ... (BEL or ST)
	ansiOSC = regexp.MustCompile(`\x1b\][^\x07]*(\x07|\x1b\\)`)
	// Other ESC sequences
	ansiOther = regexp.MustCompile(`\x1b[@-Z\\-_]`)
)

func StripANSI(s string) string {
	s = ansiOSC.ReplaceAllString(s, "")
	s = ansiCSI.ReplaceAllString(s, "")
	s = ansiOther.ReplaceAllString(s, "")
	return s
}

// ShouldStripANSI controls if we strip ANSI for terminal mirroring and/or notifications.
func ShouldStripANSI(noColor bool, mode string) bool {
	if noColor {
		return true
	}
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // auto
		return false
	}
}

func ShouldStripProgress(mode string) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // auto
		return true
	}
}

// StripProgress removes carriage-return based progress/spinner artifacts.
// It keeps only the final segment after the last carriage return and drops remaining CRs.
func StripProgress(s string) string {
	if idx := strings.LastIndex(s, "\r"); idx >= 0 {
		s = s[idx+1:]
	}
	return strings.ReplaceAll(s, "\r", "")
}
