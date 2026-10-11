package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/iSerganov/mist"
	"github.com/spf13/cobra"
)

type catchOptions struct {
	input   string
	outDir  string
	partial bool
	key     string
	timeout time.Duration
}

func newCatchCmd() *cobra.Command {
	o := &catchOptions{}
	cmd := &cobra.Command{
		Use:   "catch",
		Short: "Recover hidden text or files from an audio file or stream",
		Long: "  Scan --input for frames that decrypt under --key, printing each one\n" +
			"  as it is recovered. A file ends by itself; a live stream runs until\n" +
			"  --timeout elapses or the process is interrupted.\n\n" +
			"  Frames that carry nothing and frames meant for another key are\n" +
			"  indistinguishable, so both are skipped in silence.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCatch(cmd.Context(), o)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.input, "input", "i", "", "source to read: path, file:// or http(s):// URL")
	f.StringVarP(&o.key, "key", "k", "", "private key file")
	f.StringVar(&o.outDir, "out-dir", ".", "directory that receives recovered images, audio and files")
	f.BoolVar(&o.partial, "partial", false, "keep what arrived of a message cut short, saved with a .partial suffix")
	f.DurationVarP(&o.timeout, "timeout", "t", 0, "stop after this long (0 reads until end of stream)")
	_ = cmd.MarkFlagRequired("input")
	_ = cmd.MarkFlagRequired("key")
	return cmd
}

func runCatch(ctx context.Context, o *catchOptions) error {
	priv, err := readKey(o.key, mist.PrivateKeySize)
	if err != nil {
		return err
	}

	header("catch")
	field("source", bold(o.input), "")
	field("key", bold(o.key), "")
	field("timeout", bold(describeTimeout(o.timeout)), "")

	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	opts := []mist.CatcherOption{
		mist.WithLogger(slog.New(noteHandler{})),
		mist.WithCatchProgress(catchMeter()),
	}
	if o.partial {
		opts = append(opts, mist.WithPartial())
	}
	catcher, err := mist.NewCatcher(priv, opts...)
	if err != nil {
		return err
	}
	step("listening…")

	results, err := catcher.Listen(ctx, o.input)
	if err != nil {
		return err
	}

	found := 0
	for r := range results {
		found++
		if err := show(o.outDir, r); err != nil {
			return err
		}
	}

	success("%d frame(s) recovered · %s", found, stopReason(ctx))
	if found == 0 {
		return errors.New("no messages recovered")
	}
	return nil
}

func describeTimeout(d time.Duration) string {
	if d <= 0 {
		return "none, read to end of stream"
	}
	return d.String()
}

func stopReason(ctx context.Context) string {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "timeout reached"
	case errors.Is(ctx.Err(), context.Canceled):
		return "interrupted"
	default:
		return "end of stream"
	}
}

func show(dir string, r mist.Result) error {
	label := string(r.Payload.Data)
	if r.Payload.Type != mist.PayloadText {
		saved, err := saveRecovered(dir, r)
		if err != nil {
			return err
		}
		label = "saved " + saved
	}
	result(r.FrameIdx, len(r.Payload.Data), label)
	if r.Partial {
		warn("partial: %s of %s arrived", humanBytes(int64(len(r.Payload.Data))), humanBytes(int64(r.Total)))
	}
	return nil
}

func catchMeter() mist.ProgressFunc {
	var scan, assemble mist.Progress
	return func(p mist.Progress) {
		if p.Stage == mist.ProgressAssemble {
			assemble = p
		} else {
			scan = p
		}
		line := fmt.Sprintf("frame %d", scan.Done)
		if scan.Total > 0 {
			line += fmt.Sprintf("/%d", scan.Total)
		}
		if assemble.Total > 0 && assemble.Done < assemble.Total {
			line += fmt.Sprintf(" · receiving %s / %s", humanBytes(int64(assemble.Done)), humanBytes(int64(assemble.Total)))
		}
		meter("%s", line)
	}
}
