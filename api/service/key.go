package service

import (
	"sync/atomic"

	"github.com/photon-storage/go-photon/crypto/bls"
	"github.com/photon-storage/go-photon/crypto/interop"
	"github.com/photon-storage/go-photon/sak/guard"
)

const keySize = 10

var (
	next uint32
	sks  []bls.SecretKey
)

func init() {
	var err error
	sks, _, err = interop.DeterministicallyGenerateKeys(0, keySize)
	guard.NoError(err)
}

func nextSk() bls.SecretKey {
	n := atomic.AddUint32(&next, 1)
	return sks[(n-1)%keySize]
}
