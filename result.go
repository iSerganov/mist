package mist

// Result is emitted by Catcher whenever a stego frame decrypts and
// authenticates successfully.
type Result struct {
	Payload  Payload
	FrameIdx int64 // ordinal of the frame within this Listen session (0-based)
}
