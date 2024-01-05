package web3

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/dexidp/dex/connector"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func newConnector(t *testing.T) *web3Connector {
	log := logrus.New()

	testConfig := Config{
		InfuraID: "mockInfuraID",
	}

	conn, err := testConfig.Open("id", log)
	if err != nil {
		t.Fatal(err)
	}

	web3Conn, ok := conn.(*web3Connector)
	if !ok {
		t.Fatal(err)
	}

	return web3Conn
}

func generateWallet() (*ecdsa.PrivateKey, *common.Address, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	userAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	return privateKey, &userAddr, nil
}

func signMessage(msg string, pk *ecdsa.PrivateKey) ([]byte, common.Hash) {
	data := []byte(msg)
	hash := crypto.Keccak256Hash(data)

	signature, err := crypto.Sign(hash.Bytes(), pk)
	if err != nil {
		log.Fatal(err)
	}

	return signature, hash
}

func TestEOALogin(t *testing.T) {
	conn := newConnector(t)

	type testCase struct {
		address       string
		msg           string
		signedMessage string
		shouldErr     bool
		err           error
		identity      connector.Identity
	}

	pk, addr, err := generateWallet()
	assert.NoError(t, err)

	rawMsg := "Mock Signable Message"
	sigMsg, hash := signMessage(rawMsg, pk)

	testCases := map[string]func() testCase{
		"decode_error_signed_message": func() testCase {
			return testCase{
				address:       addr.Hex(),
				msg:           "",
				signedMessage: "",
				shouldErr:     true,
				err:           errors.New("could not decode hex string of signed nonce: empty hex string"),
			}
		},
		"v_parameter_error_signed_message": func() testCase {
			sigWithInvalidVParam := sigMsg
			sigWithInvalidVParam[64] = 100

			return testCase{
				address:       addr.Hex(),
				msg:           hash.Hex(),
				signedMessage: hexutil.Encode(sigWithInvalidVParam),
				shouldErr:     true,
				err:           fmt.Errorf("byte at index 64 of signed message should be 27 or 28: %s", hexutil.Encode(sigMsg)),
			}
		},
		"error_mismatch_address": func() testCase {
			pk2, _, err2 := generateWallet()
			assert.NoError(t, err2)

			sigMsg2, hash := signMessage(rawMsg, pk2)

			return testCase{
				address:       addr.Hex(),
				msg:           hash.Hex(),
				signedMessage: hexutil.Encode(sigMsg2),
				shouldErr:     true,
				err:           fmt.Errorf("given address and address recovered from signed nonce do not match"),
			}
		},
		"success_verify_signature": func() testCase {
			return testCase{
				address:       addr.Hex(),
				msg:           hash.Hex(),
				signedMessage: hexutil.Encode(sigMsg),
				shouldErr:     false,
				err:           nil,
				identity: connector.Identity{
					UserID:   addr.Hex(),
					Username: addr.Hex(),
				},
			}
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			tc := testCase()
			res, err := conn.Verify(tc.address, rawMsg, tc.signedMessage)
			if tc.shouldErr {
				assert.ErrorContains(t, err, tc.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.identity, res)
			}
		})
	}
}
