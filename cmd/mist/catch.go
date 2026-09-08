package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/iSerganov/mist"
	"github.com/spf13/cobra"
)

type catchOptions struct {
	input   string
	key     string
	timeout time.Duration
}

func newCatchCmd() *cobra.Command {
	o := &catchOptions{}
	cmd := &cobra.Command{
		Use:   "catch",
		Short: "Recover hidden text from an audio file or stream",
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

	catcher, err := mist.NewCatcher(priv, mist.WithLogger(slog.New(noteHandler{})))
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
		result(r.FrameIdx, string(r.Payload.Data))
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
