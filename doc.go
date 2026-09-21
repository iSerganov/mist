// Package mist is an asymmetric-key audio steganography library.
//
// A sender embeds a payload using only the recipient's X25519 public key.
// Only the matching private key can recover it. The payload lives in the
// audio itself — not container metadata — so a digital recording of a live
// stream is enough to extract the message.
//
// Output is Ogg Vorbis, where the bits go into quantized residues, or any
// lossless format the installed FFmpeg can write (FLAC, WAV, ALAC, WavPack
// and the rest), where they go into PCM sample LSBs. Formats lists them;
// WithFormat and WithCodec choose one. Carriers are whatever FFmpeg can
// decode, whatever the output is.
//
// Hybrid encryption follows the age / NaCl box shape: an ephemeral X25519
// key agreement derives a ChaCha20-Poly1305 key. Each stego frame is sealed
// independently so repeating a payload across a live stream does not produce
// a correlatable ciphertext. Unused capacity is filled with CSPRNG bytes at
// the same embedding density as a real message, so presence and absence have
// the same statistical footprint.
//
// Codec I/O goes through libav (libavformat, libavcodec, libavutil) via cgo.
// Which codecs qualify is libav's answer, not a list kept here: a format is
// usable when FFmpeg reports an encoder for it and marks it lossless.
package mist
