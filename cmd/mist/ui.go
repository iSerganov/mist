package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// out is the destination for everything the commands print. Tests swap it.
var out io.Writer = os.Stdout

var colorOn = true

// useColor turns styling off for pipes, dumb terminals and NO_COLOR.
func useColor(on bool, f *os.File) {
	if !on || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		colorOn = false
		return
	}
	info, err := f.Stat()
	colorOn = err == nil && info.Mode()&os.ModeCharDevice != 0
}

func style(code string) func(string) string {
	return func(s string) string {
		if !colorOn {
			return s
		}
		return "\x1b[" + code + "m" + s + "\x1b[0m"
	}
}

var (
	bold   = style("1")
	dim    = style("2")
	red    = style("31")
	green  = style("32")
	yellow = style("33")
	blue   = style("34")
	cyan   = style("36")
)

// printf is the single write path, so the unwritable-stdout case is
// acknowledged once rather than at every call site.
func printf(format string, args ...any) {
	_, _ = fmt.Fprintf(out, format, args...)
}

func header(title string) {
	printf("\n  %s %s %s\n\n", cyan(bold("mist")), dim("·"), bold(title))
}

// field prints an aligned label/value pair, with optional trailing note.
func field(label, value, note string) {
	pad := strings.Repeat(" ", max(0, 12-len(label)))
	line := fmt.Sprintf("  %s%s%s", dim(label), pad, value)
	if note != "" {
		line += "  " + dim(note)
	}
	printf("%s\n", line)
}

func step(msg string) {
	printf("\n  %s %s\n", blue("→"), msg)
}

func result(frame int64, text string) {
	printf("  %s %s  %s  %s\n",
		green("▸"),
		dim(fmt.Sprintf("frame %-3d", frame)),
		dim(fmt.Sprintf("%3d B", len(text))),
		bold(text),
	)
}

func success(format string, args ...any) {
	printf("\n  %s %s\n\n", green("✓"), fmt.Sprintf(format, args...))
}

func warn(format string, args ...any) {
	printf("  %s %s\n", yellow("!"), fmt.Sprintf(format, args...))
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "\n  %s %v\n\n", red("✗"), err)
}

// noteHandler renders the Catcher's diagnostics as inline warnings instead
// of structured log lines.
type noteHandler struct{}

func (noteHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h noteHandler) Handle(_ context.Context, r slog.Record) error {
	warn("%s", r.Message)
	return nil
}

func (h noteHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h noteHandler) WithGroup(string) slog.Handler      { return h }
