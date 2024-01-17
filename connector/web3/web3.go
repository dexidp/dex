// Package web3 implements logging in through a web3.js-compatible wallet
package web3

import (
	"fmt"
	"github.com/dexidp/dex/connector"
	"github.com/dexidp/dex/pkg/log"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Config struct {
	InfuraID string `json:"infuraId"`
	RPCURL   string `json:"rpcUrl"`
}

func (c *Config) Open(id string, logger log.Logger) (connector.Connector, error) {
	return &web3Connector{infuraID: c.InfuraID, rpcURL: c.RPCURL}, nil
}

type web3Connector struct {
	infuraID string
	rpcURL   string
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

	// This is the v parameter in the signature. Per the yellow paper, 27 means even and 28
	// means odd.
	if signb[64] == 27 || signb[64] == 28 {
		signb[64] -= 27
	} else if signb[64] != 0 && signb[64] != 1 {
		// We allow 0 or 1 because some non-conformant devices like Ledger use it.
		// TODO - Verify using ERC-1271
		return identity, fmt.Errorf("byte at index 64 of signed message should be 27 or 28: %s", signedMsg)
	}

	pubKey, err := crypto.SigToPub(signHash([]byte(msg)), signb)
	if err != nil {
		return identity, fmt.Errorf("failed to recover public key from signed message: %v", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	// These are byte arrays, so this is okay to do.
	if recoveredAddr != addrb {
		return identity, fmt.Errorf("given address and address recovered from signed nonce do not match")
	}

	identity.UserID = address
	identity.Username = address
	return identity, nil
}

func signHash(data []byte) []byte {
	return accounts.TextHash(data)
}
