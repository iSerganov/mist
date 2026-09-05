// Package mist is an asymmetric-key audio steganography library.
//
// A sender embeds a payload using only the recipient's X25519 public key.
// Only the matching private key can recover it. Phase 1 carries text inside
// Ogg Vorbis audio packets — not container metadata — so a digital recording
// of a live stream is enough to extract the message.
//
// Hybrid encryption follows the age / NaCl box shape: an ephemeral X25519
// key agreement derives a ChaCha20-Poly1305 key. Each stego frame is sealed
// independently so repeating a payload across a live stream does not produce
// a correlatable ciphertext. Unused capacity is filled with CSPRNG bytes at
// the same embedding density as a real message, so presence and absence have
// the same statistical footprint.
//
// Codec I/O goes through libav (libavformat, libavcodec, libavutil) via cgo.
// Coefficient-domain embedding is intentionally behind a Codec interface so
// WAV/PCM and MP3 can be added later without touching crypto or framing.
package mist
