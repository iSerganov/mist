package vorbis

// SplitExtradata unpacks the Xiph-laced Vorbis headers stored in
// libavcodec extradata: one count byte, two lacing lengths, then the
// identification, comment, and setup packets. PCM encode/decode still
// goes through libav; this only splits the blob the encoder already made.
func SplitExtradata(extra []byte) (ident, comment, setup []byte, err error) {
	if len(extra) < 3 || extra[0] != 2 {
		return nil, nil, nil, ErrBadSetup
	}
	off := 1
	n0, off, err := xiphLen(extra, off)
	if err != nil {
		return nil, nil, nil, err
	}
	n1, off, err := xiphLen(extra, off)
	if err != nil {
		return nil, nil, nil, err
	}
	if off+n0+n1 > len(extra) {
		return nil, nil, nil, ErrBadSetup
	}
	ident = extra[off : off+n0]
	comment = extra[off+n0 : off+n0+n1]
	setup = extra[off+n0+n1:]
	if len(setup) == 0 {
		return nil, nil, nil, ErrBadSetup
	}
	return ident, comment, setup, nil
}

func xiphLen(b []byte, off int) (int, int, error) {
	n := 0
	for {
		if off >= len(b) {
			return 0, 0, ErrBadSetup
		}
		n += int(b[off])
		if b[off] != 255 {
			return n, off + 1, nil
		}
		off++
	}
}
