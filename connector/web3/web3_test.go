package web3

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/dexidp/dex/connector"
)

type BkTest struct {
	t *testing.T
}

func newConnector() *web3Connector {
	log := logrus.New()

	web3Conn := &web3Connector{}
	web3Conn.logger = log

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

func signMessage(msg string, pk *ecdsa.PrivateKey) ([]byte, []byte, error) {
	data := []byte(msg)
	hash := accounts.TextHash(data)

	signature, err := crypto.Sign(hash, pk)
	if err != nil {
		return nil, nil, err
	}

	return signature, hash, nil
}

func TestEOALogin(t *testing.T) {
	conn := newConnector()
	// Create and deploy eth client
	bk := BkTest{
		t: t,
	}

	sim, auth, ctrPk, err := bk.createMockBlockchain()
	assert.NoError(t, err)

	conn.ethClient = sim

	defer func(sim *backends.SimulatedBackend) {
		err := sim.Close()
		if err != nil {
			assert.NoError(t, err)
		}
	}(sim)

	ctrAddr, _, _, err := DeployContractAuth(auth, sim, auth.From)
	assert.NoError(t, err)

	sim.Commit()
	t.Log("Contract deployed successfully to address ------", ctrAddr.Hex())
	// end

	type testCase struct {
		address       string
		msgHash       string
		signedMessage string
		shouldErr     bool
		err           error
		identity      connector.Identity
	}

	pk, addr, err := generateWallet()
	assert.NoError(t, err)

	rawMsg := "Mock Signable Message"

	testCases := map[string]func() testCase{
		"decode_error_signed_message": func() testCase {
			return testCase{
				address:       addr.Hex(),
				msgHash:       "",
				signedMessage: "",
				shouldErr:     true,
				err:           errors.New("could not decode hex string of signed nonce: empty hex string"),
			}
		},
		"v_parameter_error_signed_message": func() testCase {
			sigWithInvalidVParam, hash, err := signMessage(rawMsg, pk)
			assert.NoError(t, err)

			sigWithInvalidVParam[64] = 100

			return testCase{
				address:       addr.Hex(),
				msgHash:       hexutil.Encode(hash),
				signedMessage: hexutil.Encode(sigWithInvalidVParam),
				shouldErr:     true,
				err:           errors.New("error occurred completing login no contract code at given address"), // We get this error since ERC1271 will be called when there is a mismatch
			}
		},
		"error_mismatch_address": func() testCase {
			pk2, _, err2 := generateWallet()
			assert.NoError(t, err2)

			sigMsg2, hash, err := signMessage(rawMsg, pk2)
			assert.NoError(t, err)

			sigMsg2[64] += 27

			return testCase{
				address:       addr.Hex(),
				msgHash:       hexutil.Encode(hash),
				signedMessage: hexutil.Encode(sigMsg2),
				shouldErr:     true,
				err:           fmt.Errorf("error occurred completing login no contract code at given address"), // We get this error since ERC1271 will be called when there is a mismatch
			}
		},
		"success_verify_signature": func() testCase {
			sigMsg, hash, err := signMessage(rawMsg, pk)
			assert.NoError(t, err)

			sigMsg[64] += 27

			return testCase{
				address:       addr.Hex(),
				msgHash:       hexutil.Encode(hash),
				signedMessage: hexutil.Encode(sigMsg),
				shouldErr:     false,
				err:           nil,
				identity: connector.Identity{
					UserID:   addr.Hex(),
					Username: addr.Hex(),
				},
			}
		},
		"erc1271_success_verify_contract_signature": func() testCase {
			signature, hash, err := signMessage(rawMsg, ctrPk)
			assert.NoError(t, err)

			return testCase{
				address:       ctrAddr.Hex(),
				msgHash:       hexutil.Encode(hash),
				signedMessage: hexutil.Encode(signature),
				shouldErr:     false,
				err:           nil,
				identity: connector.Identity{
					UserID:   ctrAddr.Hex(),
					Username: ctrAddr.Hex(),
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

func TestBlockchainBackend(t *testing.T) {
	bk := BkTest{
		t: t,
	}
	conn := newConnector()

	sim, auth, pk, err := bk.createMockBlockchain()
	assert.NoError(t, err)

	conn.ethClient = sim

	defer func(sim *backends.SimulatedBackend) {
		err := sim.Close()
		if err != nil {
			assert.NoError(t, err)
		}
	}(sim)

	ctrAddr, _, _, err := DeployContractAuth(auth, sim, auth.From)
	assert.NoError(t, err)

	sim.Commit()
	t.Log("Contract deployed successfully to address", ctrAddr.Hex())

	type testCase struct {
		address       common.Address
		msgHash       []byte
		signedMessage string
		shouldErr     bool
		err           error
		identity      connector.Identity
	}

	rawMsg := "Mock Signable Message"
	testCases := map[string]func() testCase{
		"invalid_signer_error": func() testCase {
			pk2, _, err := generateWallet()
			assert.NoError(t, err)

			sg, hash, err := signMessage(rawMsg, pk2)
			assert.NoError(t, err)

			return testCase{
				address:       ctrAddr,
				msgHash:       hash,
				signedMessage: hexutil.Encode(sg),
				shouldErr:     true,
				err:           errors.New("given address and address recovered from signed message do not match"),
			}
		},
		"success_valid_signer": func() testCase {
			sg, hash, err := signMessage(rawMsg, pk)
			assert.NoError(t, err)

			return testCase{
				address:       ctrAddr,
				msgHash:       hash,
				signedMessage: hexutil.Encode(sg),
				shouldErr:     false,
				err:           nil,
				identity: connector.Identity{
					UserID:   ctrAddr.Hex(),
					Username: ctrAddr.Hex(),
				},
			}
		},
		// Note: always keep this test as the last one, if it runs before the others, ethClient will be nil
		"no_eth_client_found": func() testCase {
			conn.ethClient = nil
			var emptyByte []byte
			return testCase{
				address:       ctrAddr,
				msgHash:       emptyByte,
				signedMessage: hexutil.Encode(emptyByte),
				shouldErr:     true,
				err:           errors.New("no Ethereum client available"),
			}
		},
	}

	testList := []string{"invalid_signer_error", "success_valid_signer", "no_eth_client_found"}
	for _, name := range testList {
		testCase := testCases[name]

		t.Run(name, func(t *testing.T) {
			tc := testCase()
			res, err := conn.VerifyERC1271Signature(tc.address, tc.msgHash, tc.signedMessage)
			if tc.shouldErr {
				assert.ErrorContains(t, err, tc.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.identity, res)
			}
		})
	}
}

func (b BkTest) createMockBlockchain() (*backends.SimulatedBackend, *bind.TransactOpts, *ecdsa.PrivateKey, error) {
	pk, addr, err := generateWallet()
	if err != nil {
		b.t.Log(err, "pk err")
		return nil, nil, nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(pk, big.NewInt(1337))
	if err != nil {
		b.t.Log(err, "auth creator error")
		return nil, nil, nil, err
	}

	balance := new(big.Int)
	balance.SetString("10000000000000000000", 10) // 10 eth in wei
	blockGasLimit := uint64(8000029)

	genesisAlloc := map[common.Address]core.GenesisAccount{
		*addr: {
			Balance: balance,
		},
	}
	sim := backends.NewSimulatedBackend(genesisAlloc, blockGasLimit)

	return sim, auth, pk, nil
}
