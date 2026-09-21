package mist

import (
	"fmt"
	"sort"

	"github.com/iSerganov/mist/internal/av"
)

// DefaultFormat is the target Embed writes when none is named.
const DefaultFormat = "ogg"

// vorbisCodec is the one lossy codec Mist can embed into, because it is
// the one whose residues it can rewrite. Everything else must be lossless.
const vorbisCodec = "vorbis"

// Format is an output target Embed can write: a container, the encoder
// inside it, and the extension a file of that kind normally carries.
type Format struct {
	Container string
	Codec     string
	Ext       string
	Lossless  bool
}

// String names the format the way a caller would ask for it.
func (f Format) String() string {
	if f.Container == f.Codec {
		return f.Container
	}
	return f.Container + "/" + f.Codec
}

// LookupFormat resolves an output target the way ffmpeg's command line
// does: name is an output path whose extension names a container
// ("song.flac"), a container short name ("flac"), or an encoder name
// ("alac"); codec, when not empty, overrides the encoder the container
// would default to. An empty name means DefaultFormat.
//
// Ogg Vorbis and every lossless codec the installed FFmpeg can encode are
// supported. Any other lossy codec is ErrUnsupportedCodec: Mist can only
// hide bits it can still find afterwards, and a lossy re-encode it does
// not own the bitstream of destroys them.
func LookupFormat(name, codec string) (Format, error) {
	f, err := lookupFormat(name, codec)
	if err != nil {
		return Format{}, err
	}
	return Format{
		Container: f.Container,
		Codec:     f.CodecName,
		Ext:       f.Ext,
		Lossless:  f.Lossless,
	}, nil
}

// Formats lists every output target this FFmpeg build can write, sorted
// by container name. A container and its default encoder are reachable by
// either name, so the same target is reported once.
func Formats() []Format {
	var out []Format
	seen := map[Format]bool{}
	for _, f := range av.Formats() {
		target := Format{
			Container: f.Container,
			Codec:     f.CodecName,
			Ext:       f.Ext,
			Lossless:  f.Lossless,
		}
		if !supported(f) || seen[target] {
			continue
		}
		seen[target] = true
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Container < out[j].Container })
	return out
}

func lookupFormat(name, codec string) (av.Format, error) {
	if name == "" {
		name = codec
	}
	if name == "" {
		name = DefaultFormat
	}
	f, err := av.FindFormat(name, codec)
	if err != nil {
		return av.Format{}, fmt.Errorf("%w: %v", ErrUnsupportedCodec, err)
	}
	if !supported(f) {
		return av.Format{}, fmt.Errorf("%w: %s is lossy and Mist cannot rewrite its bitstream; "+
			"use Ogg Vorbis or any lossless codec", ErrUnsupportedCodec, f.CodecName)
	}
	return f, nil
}

func supported(f av.Format) bool {
	return f.Lossless || f.CodecName == vorbisCodec
}
