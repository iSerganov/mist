package vorbis

import "encoding/binary"

// Setup holds the three Vorbis identification / comment / setup headers
// plus the fields parsed from the identification header. Comment and
// setup are stored whole; only their packet type and "vorbis" magic are
// checked until the codebook tables exist. Audio residue decode needs
// those tables, so Residues returns ErrBadSetup until they are parsed.
type Setup struct {
	Ident      []byte
	Comment    []byte
	Codebooks  []byte
	Version    uint32
	Channels   int
	Rate       int
	BitrateMax int32
	BitrateNom int32
	BitrateMin int32
	Blocksize0 int
	Blocksize1 int
}

// ParseSetup decodes the three header packets from a fresh Ogg stream.
// Identification is parsed fully (version, channels, rate, bitrates,
// blocksizes, framing bit). Comment and setup are only checked for
// packet type and the six-byte "vorbis" magic — the rest of setup is
// Huffman/codebook data and is not walked yet.
func ParseSetup(ident, comment, setup []byte) (*Setup, error) {
	id, err := parseIdent(ident)
	if err != nil {
		return nil, err
	}
	if err := checkHeader(comment, headerComment); err != nil {
		return nil, err
	}
	if err := checkHeader(setup, headerSetup); err != nil {
		return nil, err
	}
	id.Ident = append([]byte(nil), ident...)
	id.Comment = append([]byte(nil), comment...)
	id.Codebooks = append([]byte(nil), setup...)
	return id, nil
}

func checkHeader(pkt []byte, typ byte) error {
	if len(pkt) < 1+len(magic) {
		return ErrBadPacket
	}
	if pkt[0] != typ {
		return ErrBadPacket
	}
	if string(pkt[1:7]) != magic {
		return ErrBadPacket
	}
	return nil
}

func parseIdent(pkt []byte) (*Setup, error) {
	if err := checkHeader(pkt, headerIdent); err != nil {
		return nil, err
	}
	if len(pkt) < identSize {
		return nil, ErrBadPacket
	}
	ver := binary.LittleEndian.Uint32(pkt[7:11])
	if ver != 0 {
		return nil, ErrBadSetup
	}
	ch := int(pkt[11])
	rate := int(binary.LittleEndian.Uint32(pkt[12:16]))
	if ch < 1 || rate < 1 {
		return nil, ErrBadSetup
	}
	bs := pkt[28]
	bs0 := int(bs >> 4)
	bs1 := int(bs & 0x0f)
	if bs0 < 6 || bs0 > 13 || bs1 < 6 || bs1 > 13 || bs0 > bs1 {
		return nil, ErrBadSetup
	}
	if pkt[29]&1 == 0 {
		return nil, ErrBadSetup
	}
	return &Setup{
		Version:    ver,
		Channels:   ch,
		Rate:       rate,
		BitrateMax: int32(binary.LittleEndian.Uint32(pkt[16:20])),
		BitrateNom: int32(binary.LittleEndian.Uint32(pkt[20:24])),
		BitrateMin: int32(binary.LittleEndian.Uint32(pkt[24:28])),
		Blocksize0: 1 << bs0,
		Blocksize1: 1 << bs1,
	}, nil
}
