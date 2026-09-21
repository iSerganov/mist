package main

import (
	"strings"

	"github.com/iSerganov/mist"
	"github.com/spf13/cobra"
)

func newFormatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "formats",
		Short: "List the output formats this FFmpeg build can write",
		Long: "  Ogg Vorbis, whose residues Mist rewrites, and every lossless codec\n" +
			"  the installed FFmpeg can encode. There is no list inside Mist:\n" +
			"  this is what the libraries on this machine report.\n\n" +
			"  Pick one by giving `embed --output` that extension, and name the\n" +
			"  encoder with --out-codec where a container holds more than one.",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			runFormats()
			return nil
		},
	}
}

func runFormats() {
	header("formats")
	for _, f := range mist.Formats() {
		field("."+f.Ext, bold(pad(f.Container, 10)), describeFormat(f))
	}
	printf("\n")
}

// describeFormat names the codec and how Mist will carry bits in it.
func describeFormat(f mist.Format) string {
	if f.Lossless {
		return f.Codec + " · lossless, bits in the samples"
	}
	return f.Codec + " · lossy, bits in the residues"
}

func pad(s string, n int) string {
	return s + strings.Repeat(" ", max(0, n-len(s)))
}

// capacityHint explains a full carrier in terms of the format that hit
// the limit: a lossy carrier runs out of usable residues, a lossless one
// only ever runs out of audio.
func capacityHint(f mist.Format) string {
	if f.Lossless {
		return "the carrier is too short for this message"
	}
	return "quiet or tonal audio yields too few usable residues; try a longer or " +
		"richer carrier, or a lossless --output format, which holds far more"
}
