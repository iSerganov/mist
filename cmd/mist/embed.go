package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/iSerganov/mist"
	"github.com/spf13/cobra"
)

type embedOptions struct {
	input  string
	output string
	data   string
	key    string
	codec  string
}

func newEmbedCmd() *cobra.Command {
	o := &embedOptions{}
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Hide text inside an audio file",
		Long: "  Encrypt --data for a recipient public key and embed it into --input,\n" +
			"  writing a new audio file. The output format follows the --output\n" +
			"  extension the way ffmpeg's does — .ogg, .flac, .wav and every other\n" +
			"  lossless format this FFmpeg can write. Run `mist formats` to see\n" +
			"  them, and use --out-codec where one container holds several.\n\n" +
			"  Without --key a fresh keypair is generated and both files are\n" +
			"  reported when the run finishes.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEmbed(cmd.Context(), o)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&o.input, "input", "i", "", "carrier audio file: path, file:// or http(s):// URL")
	f.StringVarP(&o.data, "data", "d", "", "text to hide")
	f.StringVarP(&o.key, "key", "k", "", "recipient public key file (a new keypair is created when omitted)")
	f.StringVarP(&o.output, "output", "o", "", "destination file; its extension picks the format (default: <input>.stego.ogg)")
	f.StringVar(&o.codec, "out-codec", "", "encoder to use when the container holds more than one (e.g. alac for .m4a)")
	_ = cmd.MarkFlagRequired("input")
	_ = cmd.MarkFlagRequired("data")
	return cmd
}

func runEmbed(ctx context.Context, o *embedOptions) error {
	format, err := mist.LookupFormat(o.output, o.codec)
	if err != nil {
		return fmt.Errorf("%w — run `mist formats` for what this FFmpeg build can write", err)
	}
	output := o.output
	if output == "" {
		output = stegoPath(o.input, format.Ext)
	}

	pub, priv, err := recipientKey(o.key)
	if err != nil {
		return err
	}

	header("embed")
	field("carrier", bold(o.input), "")
	field("payload", bold(fmt.Sprintf("%d bytes", len(o.data))), "text")
	field("output", bold(output), describeFormat(format))
	if priv != nil {
		field("recipient", bold("new keypair"), "no --key given")
	} else {
		field("recipient", bold(o.key), "")
	}
	step("re-encoding carrier and embedding…")

	started := time.Now()
	emitter, err := mist.NewEmitter(pub, mist.WithFormat(output), mist.WithCodec(o.codec))
	if err != nil {
		return err
	}
	stream, err := emitter.Embed(ctx, o.input, mist.Text(o.data))
	if errors.Is(err, mist.ErrNoCapacity) {
		return fmt.Errorf("%w — %s", err, capacityHint(format))
	}
	if err != nil {
		return err
	}
	defer func() { _ = stream.Close() }()

	written, err := writeStream(output, stream)
	if err != nil {
		return err
	}
	success("embedded into %s  %s", bold(output), dim(fmt.Sprintf("(%s in %s)",
		humanBytes(written), time.Since(started).Round(time.Millisecond))))

	if priv == nil {
		return nil
	}
	return reportNewKeys(output, pub, priv)
}

// recipientKey loads the given public key, or mints a keypair when none was
// supplied. A non-nil private key means the caller must save the pair.
func recipientKey(path string) (pub, priv []byte, err error) {
	if path != "" {
		pub, err = readKey(path, mist.PublicKeySize)
		return pub, nil, err
	}
	return mist.GenerateKeyPair()
}

func reportNewKeys(output string, pub, priv []byte) error {
	pubPath, privPath := keyPaths(output)
	if err := writeKey(pubPath, pub, publicPerm); err != nil {
		return err
	}
	if err := writeKey(privPath, priv, privatePerm); err != nil {
		return fmt.Errorf("%w — %s is unreadable without it", err, output)
	}
	field("public key", bold(pubPath), "share with senders")
	field("private key", bold(privPath), "keep secret — required to read the message")
	warn("%s", "without the private key the message cannot be recovered")
	printf("\n")
	return nil
}

func writeStream(path string, r io.Reader) (int64, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("create output: %w", err)
	}
	defer func() { _ = f.Close() }()
	n, err := io.Copy(f, r)
	if err != nil {
		return n, fmt.Errorf("write output: %w", err)
	}
	return n, f.Sync()
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGT"[exp])
}
