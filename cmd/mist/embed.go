package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/iSerganov/mist"
	"github.com/spf13/cobra"
)

type embedOptions struct {
	input  string
	output string
	data   string
	file   string
	key    string
	codec  string
	reseal bool
}

func newEmbedCmd() *cobra.Command {
	o := &embedOptions{}
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Hide text or a file inside an audio file",
		Long: "  Encrypt --data (text) or --file (an image, audio or any file) for a\n" +
			"  recipient public key and embed it into --input,\n" +
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
	f.StringVarP(&o.file, "file", "f", "", "file to hide: an image, audio or any other file")
	f.StringVarP(&o.key, "key", "k", "", "recipient public key file (a new keypair is created when omitted)")
	f.StringVarP(&o.output, "output", "o", "", "destination file; its extension picks the format (default: <input>.stego.ogg)")
	f.StringVar(&o.codec, "out-codec", "", "encoder to use when the container holds more than one (e.g. alac for .m4a)")
	f.BoolVar(&o.reseal, "reseal", false, "repeat the message in every 8 s frame so a listener who joins late still gets it; it must fit one frame")
	_ = cmd.MarkFlagRequired("input")
	cmd.MarkFlagsOneRequired("data", "file")
	cmd.MarkFlagsMutuallyExclusive("data", "file")
	return cmd
}

func runEmbed(ctx context.Context, o *embedOptions) error {
	format, err := mist.LookupFormat(o.output, o.codec)
	if err != nil {
		return fmt.Errorf("%w — run `mist formats` for what this FFmpeg build can write", err)
	}
	payload := mist.Text(o.data)
	if o.file != "" {
		if payload, err = payloadFromFile(o.file); err != nil {
			return err
		}
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
	size, kind := payloadSummary(payload)
	field("payload", bold(fmt.Sprintf("%d bytes", size)), kind)
	field("output", bold(output), describeFormat(format))
	if priv != nil {
		field("recipient", bold("new keypair"), "no --key given")
	} else {
		field("recipient", bold(o.key), "")
	}
	step("re-encoding carrier and embedding…")

	started := time.Now()
	opts := []mist.EmitterOption{
		mist.WithFormat(output), mist.WithCodec(o.codec),
		mist.WithEmitProgress(embedMeter),
	}
	if o.reseal {
		opts = append(opts, mist.WithReseal())
	}
	emitter, err := mist.NewEmitter(pub, opts...)
	if err != nil {
		return err
	}
	written, err := embedToFile(ctx, emitter, o.input, output, payload)
	if errors.Is(err, mist.ErrNoCapacity) {
		return fmt.Errorf("%w — %s", err, capacityHint(format))
	}
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

// embedMeter shows the frame count, out of the total when the carrier
// states one: a live stream has none.
func embedMeter(p mist.Progress) {
	if p.Total == 0 {
		meter("embedding frame %d", p.Done)
		return
	}
	meter("embedding frame %d/%d", p.Done, p.Total)
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

// embedToFile embeds into a temporary file beside path and renames it over
// path only once the run has succeeded. A failed run then leaves whatever
// was at path untouched, and so does a run whose output is its own input
// until the input has been read to the end.
func embedToFile(ctx context.Context, em *mist.Emitter, input, path string, payload mist.Payload) (int64, error) {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return 0, fmt.Errorf("create output: %w", err)
	}
	err = em.EmbedTo(ctx, input, f, payload)
	if err == nil {
		err = finishOutput(f)
	}
	size, _ := f.Seek(0, io.SeekEnd)
	if cerr := f.Close(); err == nil && cerr != nil {
		err = fmt.Errorf("write output: %w", cerr)
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return 0, err
	}
	return size, nil
}

// finishOutput gives the temporary file the mode os.Create would under the
// usual umask, since CreateTemp makes it private, and flushes it to disk.
func finishOutput(f *os.File) error {
	if err := f.Chmod(0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
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
