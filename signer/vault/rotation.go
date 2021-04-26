package vault

import (
	"github.com/dexidp/dex/signer"
)

func (s *Signer) RotateKey() error {
	return signer.ErrRotationNotSupported
}
