package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/iSerganov/mist"
	"github.com/spf13/cobra"
)

type estimateOptions struct {
	input  string
	output string
	codec  string
}

func newEstimateCmd() *cobra.Command {
	o := &estimateOptions{}
	cmd := &cobra.Command{
		Use:   "estimate",
		Short: "Check how much data a carrier can hold before embedding",
		Long: "  Decode --input the way `embed` would for the target named by\n" +
			"  --output/--out-codec, and report whether it holds any room for\n" +
			"  the protocol envelope and how large a text payload would fit —\n" +
			"  in one frame, and in total once embed is allowed to spread a\n" +
			"  larger message across as many frames as it needs.\n\n" +
			"  This costs what embed costs — the carrier is decoded and, for a\n" +
			"  Vorbis target, re-encoded — because capacity depends on what the\n" +
			"  target encoder actually produces, not on the source file. No key\n" +
			"  is needed: capacity does not depend on the recipient's identity.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEstimate(cmd.Context(), o)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.input, "input", "i", "", "carrier audio file: path, file:// or http(s):// URL")
	f.StringVarP(&o.output, "output", "o", "", "target container or output path whose extension picks one (default: ogg)")
	f.StringVar(&o.codec, "out-codec", "", "encoder to use when the container holds more than one (e.g. alac for .m4a)")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

func runEstimate(ctx context.Context, o *estimateOptions) error {
	format, err := mist.LookupFormat(o.output, o.codec)
	if err != nil {
		return fmt.Errorf("%w — run `mist formats` for what this FFmpeg build can write", err)
	}

	header("estimate")
	field("carrier", bold(o.input), "")
	field("target", bold(format.String()), describeFormat(format))
	step("decoding and probing capacity…")

	est, err := mist.EstimateCapacity(ctx, o.input, o.output, o.codec)
	if err != nil {
		return err
	}

	printf("\n")
	field("source", bold(describeSource(est.Source)), sourceDetails(est.Source))

	usableFrac := 0.0
	if est.Frames > 0 {
		usableFrac = float64(est.Usable) / float64(est.Frames)
	}
	printf("\n")
	field("frames", fmt.Sprintf("%s  %d/%d usable", bar(usableFrac, 24), est.Usable, est.Frames), fmt.Sprintf("%d total", est.Frames))
	field("frame cap", bold(humanBytes(int64(est.FrameCapacity))), "largest message that fits in a single frame")
	field("total cap", bold(humanBytes(int64(est.TotalCapacity))), "largest message spanning every usable frame")

	if est.TotalCapacity <= 0 {
		printf("\n")
		return fmt.Errorf("%w — %s", mist.ErrNoCapacity, capacityHint(format))
	}
	success("this carrier can be used with `mist embed`")
	return nil
}

// describeSource names the source the way describeFormat names a target:
// container/codec plus whether libav reports it as lossless.
func describeSource(s mist.Source) string {
	name := s.Codec
	if name == "" {
		name = "unknown codec"
	}
	if s.Container != "" && s.Container != s.Codec {
		name = s.Container + "/" + s.Codec
	}
	kind := "lossy"
	if s.Lossless {
		kind = "lossless"
	}
	return name + " · " + kind
}

// sourceDetails renders the rest of what libav reports about the source:
// sample rate, channel count, bitrate where the source declares one, and
// duration.
func sourceDetails(s mist.Source) string {
	parts := []string{fmt.Sprintf("%d Hz", s.SampleRate), fmt.Sprintf("%dch", s.Channels)}
	if s.Bitrate > 0 {
		parts = append(parts, fmt.Sprintf("%d kbps", s.Bitrate/1000))
	}
	if s.Duration > 0 {
		parts = append(parts, s.Duration.Round(time.Second).String())
	}
	return strings.Join(parts, " · ")
}

// bar renders a fixed-width usable/total ratio so a carrier's headroom
// reads at a glance instead of as two bare numbers.
func bar(ratio float64, width int) string {
	ratio = min(max(ratio, 0), 1)
	filled := int(ratio*float64(width) + 0.5)
	col := green
	switch {
	case ratio == 0:
		col = red
	case ratio < 0.5:
		col = yellow
	}
	return col(strings.Repeat("█", filled)) + dim(strings.Repeat("░", width-filled))
}
