package mist

// GenerateKeyPair creates a new X25519 keypair for a recipient.
// pub is shared freely with senders; priv is kept secret by the recipient.
func GenerateKeyPair() (pub, priv []byte, err error) {
	return nil, nil, errUnimplemented
}

// GenerateSigningKeyPair creates an Ed25519 keypair for optional sender
// authentication. Pass the private key to NewEmitter via WithSenderAuth.
func GenerateSigningKeyPair() (pub, priv []byte, err error) {
	return nil, nil, errUnimplemented
}
