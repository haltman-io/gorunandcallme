package app

import (
	"fmt"
	"io"
	"os"

	"github.com/haltman-io/gorunandcallme/internal/util"
)

type UI struct {
	Out     io.Writer
	Err     io.Writer
	Color   bool
	Verbose bool
	Debug   bool
}

func NewUI(noColor bool, verbose bool, debug bool) *UI {
	return &UI{
		Out:     os.Stdout,
		Err:     os.Stderr,
		Color:   !noColor,
		Verbose: verbose,
		Debug:   debug,
	}
}

func (u *UI) BannerIfAllowed(silent bool) {
	if silent {
		return
	}
	fmt.Fprintln(u.Out, PickBanner())
	fmt.Fprintln(u.Out)
	fmt.Fprintf(u.Out, "haltman.io (https://github.com/haltman-io)\n\n[codename: %s] - [release: %s]\n\n", Codename, Version)
}

func (u *UI) Info(msg string, args ...any) {
	u.printTag("INF", u.Out, msg, args...)
}

func (u *UI) Warn(msg string, args ...any) {
	u.printTag("WRN", u.Err, msg, args...)
}

func (u *UI) Error(msg string, args ...any) {
	u.printTag("ERR", u.Err, msg, args...)
}

func (u *UI) Debugf(msg string, args ...any) {
	if !u.Debug {
		return
	}
	u.printTag("DBG", u.Err, msg, args...)
}

func (u *UI) Verbosef(msg string, args ...any) {
	if !u.Verbose {
		return
	}
	u.printTag("VRB", u.Err, msg, args...)
}

func (u *UI) printTag(tag string, w io.Writer, msg string, args ...any) {
	line := fmt.Sprintf(msg, args...)
	if u.Color {
		fmt.Fprintf(w, "%s %s\n", util.ColorTag(tag), line)
		return
	}
	fmt.Fprintf(w, "[%s] %s\n", tag, line)
}
