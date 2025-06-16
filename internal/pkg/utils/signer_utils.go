package utils

import "bytes"

func SelectSigner(signers [][]byte, userWallet []byte) []byte {
	if len(signers) == 0 {
		panic("no signers")
	}
	if bytes.Equal(signers[0], userWallet) {
		return nil
	}
	return signers[0]
}
