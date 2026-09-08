// Command mist hides a text message inside an audio file and recovers it
// again with the matching private key.
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	// SIGKILL cannot be trapped; SIGINT/SIGTERM/SIGHUP cancel the context,
	// which both Embed and Listen already honour mid-frame.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			fail(err)
		}
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var noColor bool

	root := &cobra.Command{
		Use:   "mist",
		Short: "Asymmetric-key audio steganography",
		Long: "  Hide a message inside Ogg Vorbis audio using only the recipient's\n" +
			"  public key. Only the matching private key can read it back.\n\n" +
			"  The payload lives in the compressed audio itself, not in tags or\n" +
			"  container metadata, so a digital recording of a stream carries it.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			useColor(!noColor, os.Stdout)
		},
	}
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable coloured output")
	root.AddCommand(newEmbedCmd(), newCatchCmd())
	return root
}
