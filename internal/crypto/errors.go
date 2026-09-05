package crypto

import "errors"

var (
	ErrInvalidKey = errors.New("crypto: invalid key")
	ErrOpen       = errors.New("crypto: open failed")
	ErrVerify     = errors.New("crypto: signature verify failed")
)
