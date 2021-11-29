// Package web3 implements logging in through a web3.js-compatible wallet
package web3

import (
	"fmt"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Config struct{}

func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	return &Web3Connector{}, nil
}

type Web3Connector struct{}

// https://gist.github.com/dcb9/385631846097e1f59e3cba3b1d42f3ed#file-eth_sign_verify-go
func (c *Web3Connector) Verify(address, msg, signedMsg string) (identity connector.Identity, err error) {
	addrb := common.HexToAddress(address)

	signb, err := hexutil.Decode(signedMsg)
	if err != nil {
		return identity, fmt.Errorf("Could not decode hex string of signed nonce: %v", err)
	}

	if signb[64] != 27 && signb[64] != 28 {
		return identity, fmt.Errorf("Byte at index 64 of signed message should be 27 or 28: %s", signedMsg)
	}
	signb[64] -= 27

	pubKey, err := crypto.SigToPub(signHash([]byte(msg)), signb)
	if err != nil {
		return identity, fmt.Errorf("Failed to recover public key from signed message: %v", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	if recoveredAddr == addrb {
		identity.UserID = address
		identity.Username = address
		return identity, nil
	}

	return identity, fmt.Errorf("Given address and address recovered from signed nonce do not match")
}

func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}
