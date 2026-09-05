# mist

[![CI](https://github.com/iSerganov/mist/actions/workflows/ci.yml/badge.svg)](https://github.com/iSerganov/mist/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/iSerganov/mist/badges/coverage.json)](https://github.com/iSerganov/mist/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Asymmetric-key audio steganography library for Go.

Mist is built for educational purposes: a place to explore steganography,
hybrid cryptography, and bitstream-level audio embedding. Contributions
are very welcome.

A sender embeds a payload using only the recipient's public key. Only the
matching private key can recover it. Phase 1 carries text inside Ogg Vorbis
audio packets, so a digital recording of a live stream is enough to extract
the message.

This library exists for legitimate confidential communication, digital
watermarking, and steganography/steganalysis research. It is not designed
for, and should not be used for, evading detection of illegal content.

## Install

```bash
go get github.com/iSerganov/mist
```

A C compiler is required (cgo). System FFmpeg/libav is not required until
the codec layer is implemented.

## Usage

```go
pub, priv, err := mist.GenerateKeyPair()

emitter, err := mist.NewEmitter(pub)
stego, err := emitter.EmbedReader(ctx, carrier, mist.Text("hello")) // io.ReadCloser
defer stego.Close()

catcher, err := mist.NewCatcher(priv)
ch, err := catcher.Listen(ctx, "/path/to/recording.ogg")
for result := range ch {
    fmt.Println(string(result.Payload.Data))
}
```

`Emitter` also has `Embed` (path or URL) and `EmbedFile`; all three
return an `io.ReadCloser` that the caller must Close. `Catcher` has
`ListenReader` and `Extract` for a synchronous scan of an `*os.File`.
Live streams use `Listen` with an `http(s)://` URL and a cancellable
context.

## Status

Phase 1 scaffolding: public API, domain types, and the libav cgo boundary
are in place. Implementations are stubs.

See [CLAUDE.md](CLAUDE.md) for package layout and design constraints.

## Development

```bash
make test   # go test -race -count=1 ./...
make lint   # golangci-lint run --timeout=5m
```

CI (golangci-lint v2.11.4, race tests, coverage badge) matches [gocue](https://github.com/iSerganov/gocue).

## Contributing

Issues, design notes, and pull requests are very welcome — this project
is meant to be learned from and built with.
