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

// ContractAuthMetaData contains all meta data concerning the ContractAuth contract.
var ContractAuthMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_expectedAddr\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes4\",\"name\":\"isValid\",\"type\":\"bytes4\"}],\"name\":\"ValidatedSigner\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_hash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"_signature\",\"type\":\"bytes\"}],\"name\":\"isValidSignature\",\"outputs\":[{\"internalType\":\"bytes4\",\"name\":\"\",\"type\":\"bytes4\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_expectedAddr\",\"type\":\"address\"}],\"name\":\"setExpectedSigner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50604051610e0b380380610e0b833981810160405281019061003291906100db565b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050610108565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006100a88261007d565b9050919050565b6100b88161009d565b81146100c357600080fd5b50565b6000815190506100d5816100af565b92915050565b6000602082840312156100f1576100f0610078565b5b60006100ff848285016100c6565b91505092915050565b610cf4806101176000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c80631626ba7e1461003b5780639b7995e81461006b575b600080fd5b61005560048036038101906100509190610633565b610087565b60405161006291906106ce565b60405180910390f35b61008560048036038101906100809190610747565b61021c565b005b6000808383604081811061009e5761009d610774565b5b9050013560f81c60f81b60f81c9050601b8160ff16106100f3576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016100ea90610826565b60405180910390fd5b601b816101009190610882565b9050600084846040516020016101179291906108f6565b6040516020818303038152906040529050816040516020016101399190610945565b604051602081830303815290604052610151906109c9565b8160408151811061016557610164610774565b5b60200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a90535060006101a087836102ce565b905060008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff160361020757631626ba7e60e01b9350505050610215565b63ffffffff60e01b93505050505b9392505050565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff160361028b576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161028290610aa2565b60405180910390fd5b806000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b60008060006102dd85856102f5565b915091506102ea81610346565b819250505092915050565b60008060418351036103365760008060006020860151925060408601519150606086015160001a905061032a878285856104ac565b9450945050505061033f565b60006002915091505b9250929050565b6000600481111561035a57610359610ac2565b5b81600481111561036d5761036c610ac2565b5b03156104a9576001600481111561038757610386610ac2565b5b81600481111561039a57610399610ac2565b5b036103da576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103d190610b3d565b60405180910390fd5b600260048111156103ee576103ed610ac2565b5b81600481111561040157610400610ac2565b5b03610441576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161043890610ba9565b60405180910390fd5b6003600481111561045557610454610ac2565b5b81600481111561046857610467610ac2565b5b036104a8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161049f90610c3b565b60405180910390fd5b5b50565b6000807f7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a08360001c11156104e7576000600391509150610585565b60006001878787876040516000815260200160405260405161050c9493929190610c79565b6020604051602081039080840390855afa15801561052e573d6000803e3d6000fd5b505050602060405103519050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff160361057c57600060019250925050610585565b80600092509250505b94509492505050565b600080fd5b600080fd5b6000819050919050565b6105ab81610598565b81146105b657600080fd5b50565b6000813590506105c8816105a2565b92915050565b600080fd5b600080fd5b600080fd5b60008083601f8401126105f3576105f26105ce565b5b8235905067ffffffffffffffff8111156106105761060f6105d3565b5b60208301915083600182028301111561062c5761062b6105d8565b5b9250929050565b60008060006040848603121561064c5761064b61058e565b5b600061065a868287016105b9565b935050602084013567ffffffffffffffff81111561067b5761067a610593565b5b610687868287016105dd565b92509250509250925092565b60007fffffffff0000000000000000000000000000000000000000000000000000000082169050919050565b6106c881610693565b82525050565b60006020820190506106e360008301846106bf565b92915050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000610714826106e9565b9050919050565b61072481610709565b811461072f57600080fd5b50565b6000813590506107418161071b565b92915050565b60006020828403121561075d5761075c61058e565b5b600061076b84828501610732565b91505092915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082825260208201905092915050565b7f496e76616c696420636f6e7472616374207369676e61747572652070726f766960008201527f6465640000000000000000000000000000000000000000000000000000000000602082015250565b60006108106023836107a3565b915061081b826107b4565b604082019050919050565b6000602082019050818103600083015261083f81610803565b9050919050565b600060ff82169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600061088d82610846565b915061089883610846565b9250828201905060ff8111156108b1576108b0610853565b5b92915050565b600081905092915050565b82818337600083830152505050565b60006108dd83856108b7565b93506108ea8385846108c2565b82840190509392505050565b60006109038284866108d1565b91508190509392505050565b60008160f81b9050919050565b60006109278261090f565b9050919050565b61093f61093a82610846565b61091c565b82525050565b6000610951828461092e565b60018201915081905092915050565b600081519050919050565b6000819050602082019050919050565b60007fff0000000000000000000000000000000000000000000000000000000000000082169050919050565b60006109b3825161097b565b80915050919050565b600082821b905092915050565b60006109d482610960565b826109de8461096b565b90506109e9816109a7565b92506001821015610a2957610a247fff00000000000000000000000000000000000000000000000000000000000000836001036008026109bc565b831692505b5050919050565b7f6578706563746564207369676e65722063616e6e6f742062652073657420746f60008201527f2030206164647265737300000000000000000000000000000000000000000000602082015250565b6000610a8c602a836107a3565b9150610a9782610a30565b604082019050919050565b60006020820190508181036000830152610abb81610a7f565b9050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602160045260246000fd5b7f45434453413a20696e76616c6964207369676e61747572650000000000000000600082015250565b6000610b276018836107a3565b9150610b3282610af1565b602082019050919050565b60006020820190508181036000830152610b5681610b1a565b9050919050565b7f45434453413a20696e76616c6964207369676e6174757265206c656e67746800600082015250565b6000610b93601f836107a3565b9150610b9e82610b5d565b602082019050919050565b60006020820190508181036000830152610bc281610b86565b9050919050565b7f45434453413a20696e76616c6964207369676e6174757265202773272076616c60008201527f7565000000000000000000000000000000000000000000000000000000000000602082015250565b6000610c256022836107a3565b9150610c3082610bc9565b604082019050919050565b60006020820190508181036000830152610c5481610c18565b9050919050565b610c6481610598565b82525050565b610c7381610846565b82525050565b6000608082019050610c8e6000830187610c5b565b610c9b6020830186610c6a565b610ca86040830185610c5b565b610cb56060830184610c5b565b9594505050505056fea26469706673582212204126ca1ef96d56dad22f291a477265815c97d1f0c016e65bc1560467a968b8c964736f6c63430008140033",
}

// ContractAuthABI is the input ABI used to generate the binding from.
// Deprecated: Use ContractAuthMetaData.ABI instead.
var ContractAuthABI = ContractAuthMetaData.ABI

// ContractAuthBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ContractAuthMetaData.Bin instead.
var ContractAuthBin = ContractAuthMetaData.Bin

// DeployContractAuth deploys a new Ethereum contract, binding an instance of ContractAuth to it.
func DeployContractAuth(auth *bind.TransactOpts, backend bind.ContractBackend, _expectedAddr common.Address) (common.Address, *types.Transaction, *ContractAuth, error) {
	parsed, err := ContractAuthMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ContractAuthBin), backend, _expectedAddr)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ContractAuth{ContractAuthCaller: ContractAuthCaller{contract: contract}, ContractAuthTransactor: ContractAuthTransactor{contract: contract}, ContractAuthFilterer: ContractAuthFilterer{contract: contract}}, nil
}

// ContractAuth is an auto generated Go binding around an Ethereum contract.
type ContractAuth struct {
	ContractAuthCaller     // Read-only binding to the contract
	ContractAuthTransactor // Write-only binding to the contract
	ContractAuthFilterer   // Log filterer for contract events
}

// ContractAuthCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContractAuthCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractAuthTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContractAuthTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractAuthFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContractAuthFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractAuthSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContractAuthSession struct {
	Contract     *ContractAuth     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ContractAuthCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContractAuthCallerSession struct {
	Contract *ContractAuthCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// ContractAuthTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContractAuthTransactorSession struct {
	Contract     *ContractAuthTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// ContractAuthRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContractAuthRaw struct {
	Contract *ContractAuth // Generic contract binding to access the raw methods on
}

// ContractAuthCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContractAuthCallerRaw struct {
	Contract *ContractAuthCaller // Generic read-only contract binding to access the raw methods on
}

// ContractAuthTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContractAuthTransactorRaw struct {
	Contract *ContractAuthTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContractAuth creates a new instance of ContractAuth, bound to a specific deployed contract.
func NewContractAuth(address common.Address, backend bind.ContractBackend) (*ContractAuth, error) {
	contract, err := bindContractAuth(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ContractAuth{ContractAuthCaller: ContractAuthCaller{contract: contract}, ContractAuthTransactor: ContractAuthTransactor{contract: contract}, ContractAuthFilterer: ContractAuthFilterer{contract: contract}}, nil
}

// NewContractAuthCaller creates a new read-only instance of ContractAuth, bound to a specific deployed contract.
func NewContractAuthCaller(address common.Address, caller bind.ContractCaller) (*ContractAuthCaller, error) {
	contract, err := bindContractAuth(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContractAuthCaller{contract: contract}, nil
}

// NewContractAuthTransactor creates a new write-only instance of ContractAuth, bound to a specific deployed contract.
func NewContractAuthTransactor(address common.Address, transactor bind.ContractTransactor) (*ContractAuthTransactor, error) {
	contract, err := bindContractAuth(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContractAuthTransactor{contract: contract}, nil
}

// NewContractAuthFilterer creates a new log filterer instance of ContractAuth, bound to a specific deployed contract.
func NewContractAuthFilterer(address common.Address, filterer bind.ContractFilterer) (*ContractAuthFilterer, error) {
	contract, err := bindContractAuth(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContractAuthFilterer{contract: contract}, nil
}

// bindContractAuth binds a generic wrapper to an already deployed contract.
func bindContractAuth(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ContractAuthMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ContractAuth *ContractAuthRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ContractAuth.Contract.ContractAuthCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ContractAuth *ContractAuthRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ContractAuth.Contract.ContractAuthTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ContractAuth *ContractAuthRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ContractAuth.Contract.ContractAuthTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ContractAuth *ContractAuthCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ContractAuth.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ContractAuth *ContractAuthTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ContractAuth.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ContractAuth *ContractAuthTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ContractAuth.Contract.contract.Transact(opts, method, params...)
}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 _hash, bytes _signature) view returns(bytes4)
func (_ContractAuth *ContractAuthCaller) IsValidSignature(opts *bind.CallOpts, _hash [32]byte, _signature []byte) ([4]byte, error) {
	var out []interface{}
	err := _ContractAuth.contract.Call(opts, &out, "isValidSignature", _hash, _signature)

	if err != nil {
		return *new([4]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([4]byte)).(*[4]byte)

	return out0, err

}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 _hash, bytes _signature) view returns(bytes4)
func (_ContractAuth *ContractAuthSession) IsValidSignature(_hash [32]byte, _signature []byte) ([4]byte, error) {
	return _ContractAuth.Contract.IsValidSignature(&_ContractAuth.CallOpts, _hash, _signature)
}

// IsValidSignature is a free data retrieval call binding the contract method 0x1626ba7e.
//
// Solidity: function isValidSignature(bytes32 _hash, bytes _signature) view returns(bytes4)
func (_ContractAuth *ContractAuthCallerSession) IsValidSignature(_hash [32]byte, _signature []byte) ([4]byte, error) {
	return _ContractAuth.Contract.IsValidSignature(&_ContractAuth.CallOpts, _hash, _signature)
}

// SetExpectedSigner is a paid mutator transaction binding the contract method 0x9b7995e8.
//
// Solidity: function setExpectedSigner(address _expectedAddr) returns()
func (_ContractAuth *ContractAuthTransactor) SetExpectedSigner(opts *bind.TransactOpts, _expectedAddr common.Address) (*types.Transaction, error) {
	return _ContractAuth.contract.Transact(opts, "setExpectedSigner", _expectedAddr)
}

// SetExpectedSigner is a paid mutator transaction binding the contract method 0x9b7995e8.
//
// Solidity: function setExpectedSigner(address _expectedAddr) returns()
func (_ContractAuth *ContractAuthSession) SetExpectedSigner(_expectedAddr common.Address) (*types.Transaction, error) {
	return _ContractAuth.Contract.SetExpectedSigner(&_ContractAuth.TransactOpts, _expectedAddr)
}

// SetExpectedSigner is a paid mutator transaction binding the contract method 0x9b7995e8.
//
// Solidity: function setExpectedSigner(address _expectedAddr) returns()
func (_ContractAuth *ContractAuthTransactorSession) SetExpectedSigner(_expectedAddr common.Address) (*types.Transaction, error) {
	return _ContractAuth.Contract.SetExpectedSigner(&_ContractAuth.TransactOpts, _expectedAddr)
}

// ContractAuthValidatedSignerIterator is returned from FilterValidatedSigner and is used to iterate over the raw logs and unpacked data for ValidatedSigner events raised by the ContractAuth contract.
type ContractAuthValidatedSignerIterator struct {
	Event *ContractAuthValidatedSigner // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ContractAuthValidatedSignerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ContractAuthValidatedSigner)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ContractAuthValidatedSigner)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ContractAuthValidatedSignerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ContractAuthValidatedSignerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ContractAuthValidatedSigner represents a ValidatedSigner event raised by the ContractAuth contract.
type ContractAuthValidatedSigner struct {
	Owner   common.Address
	Signer  common.Address
	IsValid [4]byte
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterValidatedSigner is a free log retrieval operation binding the contract event 0xb0a1cb3375368319eda37d9ea023fe4d12b0185fed7e4022ff1729b4bf40b40b.
//
// Solidity: event ValidatedSigner(address indexed owner, address indexed signer, bytes4 indexed isValid)
func (_ContractAuth *ContractAuthFilterer) FilterValidatedSigner(opts *bind.FilterOpts, owner []common.Address, signer []common.Address, isValid [][4]byte) (*ContractAuthValidatedSignerIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var signerRule []interface{}
	for _, signerItem := range signer {
		signerRule = append(signerRule, signerItem)
	}
	var isValidRule []interface{}
	for _, isValidItem := range isValid {
		isValidRule = append(isValidRule, isValidItem)
	}

	logs, sub, err := _ContractAuth.contract.FilterLogs(opts, "ValidatedSigner", ownerRule, signerRule, isValidRule)
	if err != nil {
		return nil, err
	}
	return &ContractAuthValidatedSignerIterator{contract: _ContractAuth.contract, event: "ValidatedSigner", logs: logs, sub: sub}, nil
}

// WatchValidatedSigner is a free log subscription operation binding the contract event 0xb0a1cb3375368319eda37d9ea023fe4d12b0185fed7e4022ff1729b4bf40b40b.
//
// Solidity: event ValidatedSigner(address indexed owner, address indexed signer, bytes4 indexed isValid)
func (_ContractAuth *ContractAuthFilterer) WatchValidatedSigner(opts *bind.WatchOpts, sink chan<- *ContractAuthValidatedSigner, owner []common.Address, signer []common.Address, isValid [][4]byte) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var signerRule []interface{}
	for _, signerItem := range signer {
		signerRule = append(signerRule, signerItem)
	}
	var isValidRule []interface{}
	for _, isValidItem := range isValid {
		isValidRule = append(isValidRule, isValidItem)
	}

	logs, sub, err := _ContractAuth.contract.WatchLogs(opts, "ValidatedSigner", ownerRule, signerRule, isValidRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ContractAuthValidatedSigner)
				if err := _ContractAuth.contract.UnpackLog(event, "ValidatedSigner", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseValidatedSigner is a log parse operation binding the contract event 0xb0a1cb3375368319eda37d9ea023fe4d12b0185fed7e4022ff1729b4bf40b40b.
//
// Solidity: event ValidatedSigner(address indexed owner, address indexed signer, bytes4 indexed isValid)
func (_ContractAuth *ContractAuthFilterer) ParseValidatedSigner(log types.Log) (*ContractAuthValidatedSigner, error) {
	event := new(ContractAuthValidatedSigner)
	if err := _ContractAuth.contract.UnpackLog(event, "ValidatedSigner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
