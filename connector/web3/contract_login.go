// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package web3

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// Erc1271MetaData contains all meta data concerning the Erc1271 contract.
var Erc1271MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"isValidSignature\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"magicValue\",\"type\":\"bytes4\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// Erc1271ABI is the input ABI used to generate the binding from.
// Deprecated: Use Erc1271MetaData.ABI instead.
var Erc1271ABI = Erc1271MetaData.ABI

// Erc1271 is an auto generated Go binding around an Ethereum contract.
type Erc1271 struct {
	Erc1271Caller     // Read-only binding to the contract
	Erc1271Transactor // Write-only binding to the contract
	Erc1271Filterer   // Log filterer for contract events
}

// Erc1271Caller is an auto generated read-only Go binding around an Ethereum contract.
type Erc1271Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc1271Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Erc1271Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc1271Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Erc1271Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc1271Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Erc1271Session struct {
	Contract     *Erc1271          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Erc1271CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Erc1271CallerSession struct {
	Contract *Erc1271Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// Erc1271TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Erc1271TransactorSession struct {
	Contract     *Erc1271Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// Erc1271Raw is an auto generated low-level Go binding around an Ethereum contract.
type Erc1271Raw struct {
	Contract *Erc1271 // Generic contract binding to access the raw methods on
}

// Erc1271CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Erc1271CallerRaw struct {
	Contract *Erc1271Caller // Generic read-only contract binding to access the raw methods on
}

// Erc1271TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Erc1271TransactorRaw struct {
	Contract *Erc1271Transactor // Generic write-only contract binding to access the raw methods on
}

// NewErc1271 creates a new instance of Erc1271, bound to a specific deployed contract.
func NewErc1271(address common.Address, backend bind.ContractBackend) (*Erc1271, error) {
	contract, err := bindErc1271(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Erc1271{Erc1271Caller: Erc1271Caller{contract: contract}, Erc1271Transactor: Erc1271Transactor{contract: contract}, Erc1271Filterer: Erc1271Filterer{contract: contract}}, nil
}

// NewErc1271Caller creates a new read-only instance of Erc1271, bound to a specific deployed contract.
func NewErc1271Caller(address common.Address, caller bind.ContractCaller) (*Erc1271Caller, error) {
	contract, err := bindErc1271(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Erc1271Caller{contract: contract}, nil
}

// NewErc1271Transactor creates a new write-only instance of Erc1271, bound to a specific deployed contract.
func NewErc1271Transactor(address common.Address, transactor bind.ContractTransactor) (*Erc1271Transactor, error) {
	contract, err := bindErc1271(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Erc1271Transactor{contract: contract}, nil
}

// NewErc1271Filterer creates a new log filterer instance of Erc1271, bound to a specific deployed contract.
func NewErc1271Filterer(address common.Address, filterer bind.ContractFilterer) (*Erc1271Filterer, error) {
	contract, err := bindErc1271(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Erc1271Filterer{contract: contract}, nil
}

// bindErc1271 binds a generic wrapper to an already deployed contract.
func bindErc1271(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Erc1271MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc1271 *Erc1271Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc1271.Contract.Erc1271Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc1271 *Erc1271Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc1271.Contract.Erc1271Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc1271 *Erc1271Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc1271.Contract.Erc1271Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc1271 *Erc1271CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc1271.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc1271 *Erc1271TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc1271.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc1271 *Erc1271TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc1271.Contract.contract.Transact(opts, method, params...)
}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 hash, bytes signature) view returns(bytes4 magicValue)
func (_Erc1271 *Erc1271Caller) IsValidSignature(opts *bind.CallOpts, hash [32]byte, signature []byte) ([4]byte, error) {
	var out []interface{}
	err := _Erc1271.contract.Call(opts, &out, "isValidSignature", hash, signature)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 hash, bytes signature) view returns(bytes4 magicValue)
func (_Erc1271 *Erc1271Session) IsValidSignature(hash [32]byte, signature []byte) ([4]byte, error) {
	return _Erc1271.Contract.IsValidSignature(&_Erc1271.CallOpts, hash, signature)
}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 hash, bytes signature) view returns(bytes4 magicValue)
func (_Erc1271 *Erc1271CallerSession) IsValidSignature(hash [32]byte, signature []byte) ([4]byte, error) {
	return _Erc1271.Contract.IsValidSignature(&_Erc1271.CallOpts, hash, signature)
}
