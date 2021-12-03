// Package web3 implements logging in through a web3.js-compatible wallet
package web3

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
)

type Config struct {
	InfuraID string `json:"infuraId"`
}

func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	return &web3Connector{infuraID: c.InfuraID}, nil
}

type web3Connector struct {
	infuraID string
}

func (c *web3Connector) InfuraID() string {
	return c.infuraID
}

// https://gist.github.com/dcb9/385631846097e1f59e3cba3b1d42f3ed#file-eth_sign_verify-go
func (c *web3Connector) Verify(address, msg, signedMsg string) (identity connector.Identity, err error) {
	addrb := common.HexToAddress(address)

	signb, err := hexutil.Decode(signedMsg)
	if err != nil {
		return identity, fmt.Errorf("could not decode hex string of signed nonce: %v", err)
	}

	if signb[64] != 27 && signb[64] != 28 {
		return identity, fmt.Errorf("byte at index 64 of signed message should be 27 or 28: %s", signedMsg)
	}
	signb[64] -= 27

	pubKey, err := crypto.SigToPub(signHash([]byte(msg)), signb)
	if err != nil {
		return identity, fmt.Errorf("failed to recover public key from signed message: %v", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	if recoveredAddr == addrb {
		identity.UserID = address
		identity.Username = address
		return identity, nil
	}

	return identity, fmt.Errorf("given address and address recovered from signed nonce do not match")
}

func (c *web3Connector) Refresh(ctx context.Context, s connector.Scopes, identity connector.Identity) (connector.Identity, error) {
	return identity, nil
}

func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}
