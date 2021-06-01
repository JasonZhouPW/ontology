/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	common4 "github.com/ethereum/go-ethereum/common"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ontio/ontology/smartcontract/service/evm"

	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology/account"
	"github.com/ontio/ontology/cmd/utils"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/constants"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/genesis"
	"github.com/ontio/ontology/core/ledger"
	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/signature"
	"github.com/ontio/ontology/core/store/ledgerstore"
	"github.com/ontio/ontology/core/types"
	utils2 "github.com/ontio/ontology/core/utils"
	"github.com/ontio/ontology/events"
	common2 "github.com/ontio/ontology/http/base/common"
	"github.com/ontio/ontology/smartcontract/service/wasmvm"
	"github.com/ontio/ontology/smartcontract/states"
	vmtypes "github.com/ontio/ontology/vm/neovm/types"
	common3 "github.com/ontio/ontology/wasmtest/common"
	"github.com/ontio/wagon/exec"
	"github.com/ontio/wagon/wasm"
)

const contractDir = "./test-contract"
const testcaseMethod = "testcase"
const WingABI = "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

func NewDeployWasmContract(signer *account.Account, code []byte) (*types.Transaction, error) {
	mutable, err := utils.NewDeployCodeTransaction(0, 100000000, code, payload.WASMVM_TYPE, "name", "version",
		"author", "email", "desc")
	if err != nil {
		return nil, err
	}
	err = utils.SignTransaction(signer, mutable)
	if err != nil {
		return nil, err
	}
	tx, err := mutable.IntoImmutable()
	return tx, err
}

func NewDeployNeoContract(signer *account.Account, code []byte) (*types.Transaction, error) {
	mutable, err := utils.NewDeployCodeTransaction(0, 100000000, code, payload.NEOVM_TYPE, "name", "version",
		"author", "email", "desc")
	if err != nil {
		return nil, err
	}
	err = utils.SignTransaction(signer, mutable)
	if err != nil {
		return nil, err
	}
	tx, err := mutable.IntoImmutable()
	return tx, err
}

func NewDeployEvmContract(testPrivateKey *ecdsa.PrivateKey, code []byte, jsonABI string, params ...interface{}) (*types2.Transaction, error) {
	chainId := big.NewInt(0)
	opts, err := bind.NewKeyedTransactorWithChainID(testPrivateKey, chainId)
	opts.GasPrice = big.NewInt(0)
	opts.Nonce = big.NewInt(0)

	checkErr(err)
	parsed, err := abi.JSON(strings.NewReader(jsonABI))
	checkErr(err)
	input, err := parsed.Pack("", params...)
	checkErr(err)
	input = append(code, input...)
	deployTx := types2.NewContractCreation(opts.Nonce.Uint64(), opts.Value, opts.GasLimit, opts.GasPrice, input)
	signedTx, err := opts.Signer(opts.From, deployTx)
	checkErr(err)
	return signedTx, err
}

func GenNeoTextCaseTransaction(contract common.Address, database *ledger.Ledger) [][]common3.TestCase {
	params := make([]interface{}, 0)
	method := string("testcase")
	// neovm entry api is def Main(method, args). and testcase method api need no other args, so pass a random args to entry api.
	operation := 1
	params = append(params, method)
	params = append(params, operation)
	tx, err := common2.NewNeovmInvokeTransaction(0, 100000000, contract, params)
	imt, err := tx.IntoImmutable()
	if err != nil {
		panic(err)
	}
	res, err := database.PreExecuteContract(imt)
	if err != nil {
		panic(err)
	}

	ret := res.Result.(string)
	jsonCase, err := common.HexToBytes(ret)

	if err != nil {
		panic(err)
	}
	if len(jsonCase) == 0 {
		panic("failed to get testcase data from contract")
	}
	var testCase [][]common3.TestCase
	err = json.Unmarshal([]byte(jsonCase), &testCase)
	if err != nil {
		panic("failed Unmarshal")
	}
	return testCase
}

func ExactTestCase(code []byte) [][]common3.TestCase {
	m, err := wasm.ReadModule(bytes.NewReader(code), func(name string) (*wasm.Module, error) {
		switch name {
		case "env":
			return wasmvm.NewHostModule(), nil
		}
		return nil, fmt.Errorf("module %q unknown", name)
	})
	checkErr(err)

	compiled, err := exec.CompileModule(m)
	checkErr(err)

	vm, err := exec.NewVMWithCompiled(compiled, 10*1024*1024)
	checkErr(err)

	param := common.NewZeroCopySink(nil)
	param.WriteString(testcaseMethod)
	host := &wasmvm.Runtime{Input: param.Bytes()}
	vm.HostData = host
	vm.RecoverPanic = true
	envGasLimit := uint64(100000000000000)
	envExecStep := uint64(100000000000000)
	vm.ExecMetrics = &exec.Gas{GasLimit: &envGasLimit, GasPrice: 0, GasFactor: 5, ExecStep: &envExecStep}
	vm.CallStackDepth = 1024

	entry := compiled.RawModule.Export.Entries["invoke"]
	index := int64(entry.Index)
	_, err = vm.ExecCode(index)
	checkErr(err)

	var testCase [][]common3.TestCase
	source := common.NewZeroCopySource(host.Output)
	jsonCase, _, _, _ := source.NextString()

	if len(jsonCase) == 0 {
		panic("failed to get testcase data from contract")
	}

	err = json.Unmarshal([]byte(jsonCase), &testCase)
	checkErr(err)

	return testCase
}

type Item struct {
	File     string
	Contract []byte
}

func LoadContracts(dir string) ([]Item, error) {
	contracts := make([]Item, 0)
	fnames, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return nil, err
	}

	sort.Strings(fnames)

	for _, name := range fnames {
		if !(strings.HasSuffix(name, ".wasm") || strings.HasSuffix(name, ".avm") ||
			strings.HasSuffix(name, ".evm")) {
			continue
		}
		raw, err := ioutil.ReadFile(name)
		if err != nil {
			return nil, err
		}
		code, _ := hex.DecodeString(string(raw))
		con := Item{
			File:     path.Base(name),
			Contract: code,
		}
		contracts = append(contracts, con)
	}

	return contracts, nil
}

func init() {
	log.InitLog(log.InfoLog, log.PATH, log.Stdout)
	runtime.GOMAXPROCS(4)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func execTxGasTest(tx *types.Transaction, database *ledger.Ledger) {
	//
	res_jit, err := database.GetStore().(*ledgerstore.LedgerStoreImp).PreExecuteContractWithParam(tx, ledgerstore.PrexecuteParam{
		JitMode:    true,
		WasmFactor: 1,
		MinGas:     false,
	})
	checkErr(err)

	res_inter, err := database.GetStore().(*ledgerstore.LedgerStoreImp).PreExecuteContractWithParam(tx, ledgerstore.PrexecuteParam{
		JitMode:    false,
		WasmFactor: 1,
		MinGas:     false,
	})
	checkErr(err)

	assertEq(res_jit, res_inter)

	//
	res_jit, err = database.GetStore().(*ledgerstore.LedgerStoreImp).PreExecuteContractWithParam(tx, ledgerstore.PrexecuteParam{
		JitMode:    true,
		WasmFactor: config.DEFAULT_WASM_GAS_FACTOR,
		MinGas:     false,
	})
	checkErr(err)

	res_inter, err = database.GetStore().(*ledgerstore.LedgerStoreImp).PreExecuteContractWithParam(tx, ledgerstore.PrexecuteParam{
		JitMode:    false,
		WasmFactor: config.DEFAULT_WASM_GAS_FACTOR,
		MinGas:     false,
	})
	checkErr(err)
	assertEq(res_jit, res_inter)
}

func execTxCheckRes(tx *types.Transaction, testCase common3.TestCase, database *ledger.Ledger, addr common.Address, acct *account.Account) {
	execTxGasTest(tx, database)

	res, err := database.PreExecuteContract(tx)
	checkErr(err)
	log.Infof("testcase consume gas: %d", res.Gas)

	height := database.GetCurrentBlockHeight()
	header, err := database.GetHeaderByHeight(height)
	checkErr(err)
	blockTime := header.Timestamp + 1

	execEnv := ExecEnv{Time: blockTime, Height: height + 1, Tx: tx, BlockHash: header.Hash(), Contract: addr}
	checkExecResult(testCase, res, execEnv)

	block, _ := makeBlock(acct, []*types.Transaction{tx})
	err = database.AddBlock(block, nil, common.UINT256_EMPTY)
	checkErr(err)
	fmt.Println("execTxCheckRes success ", testCase.Method)
}

var testPrivateKey *ecdsa.PrivateKey
var testEthAddr common4.Address

func main() {
	datadir := "testdata"
	err := os.RemoveAll(datadir)
	defer func() {
		_ = os.RemoveAll(datadir)
		_ = os.RemoveAll(log.PATH)
	}()
	checkErr(err)
	log.Trace("Node version: ", config.Version)

	acct := account.NewAccount("")
	buf := keypair.SerializePublicKey(acct.PublicKey)
	config.DefConfig.Genesis.ConsensusType = "solo"
	config.DefConfig.Genesis.SOLO.GenBlockTime = 3
	config.DefConfig.Genesis.SOLO.Bookkeepers = []string{hex.EncodeToString(buf)}
	config.DefConfig.P2PNode.NetworkId = 0

	bookkeepers := []keypair.PublicKey{acct.PublicKey}
	//Init event hub
	events.Init()

	log.Info("1. Loading the Ledger")
	database, err := ledger.NewLedger(datadir, 1000000)
	checkErr(err)
	ledger.DefLedger = database
	genblock, err := genesis.BuildGenesisBlock(bookkeepers, config.DefConfig.Genesis)
	checkErr(err)
	err = database.Init(bookkeepers, genblock)
	checkErr(err)

	log.Info("loading wasm contract")
	contract, err := LoadContracts(contractDir)
	checkErr(err)

	testPrivateKeyStr := "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"

	testPrivateKey, err = crypto.HexToECDSA(testPrivateKeyStr)
	checkErr(err)
	testEthAddr = crypto.PubkeyToAddress(testPrivateKey.PublicKey)

	log.Infof("deploying %d wasm contracts", len(contract))
	txes := make([]*types.Transaction, 0, len(contract))
	for _, item := range contract {
		file := item.File
		cont := item.Contract
		var tx *types.Transaction
		var err error
		if strings.HasSuffix(file, ".wasm") {
			tx, err = NewDeployWasmContract(acct, cont)
		} else if strings.HasSuffix(file, ".avm") {
			tx, err = NewDeployNeoContract(acct, cont)
		} else if strings.HasSuffix(file, ".evm") {
			ethtx, err := NewDeployEvmContract(testPrivateKey, cont, WingABI)
			checkErr(err)
			tx, err = types.TransactionFromEIP155(ethtx)
			checkErr(err)
			_, err = tx.GetEIP155Tx()
			checkErr(err)
		}
		checkErr(err)
		res, err := database.PreExecuteContract(tx)
		log.Infof("deploy %s consume gas: %d, %s", file, res.Gas, JsonString(res))
		checkErr(err)
		txes = append(txes, tx)
	}

	block, _ := makeBlock(acct, txes)
	err = database.AddBlock(block, nil, common.UINT256_EMPTY)
	checkErr(err)

	keyAddr := crypto.PubkeyToAddress(testPrivateKey.PublicKey)

	addrMap := make([]common3.ConAddr, 0)
	var bridge common3.ConAddr
	var wingErc20 common3.ConAddr
	var wingOep4 common3.ConAddr
	for _, item := range contract {
		file := item.File
		code := item.Contract
		if strings.HasSuffix(file, ".evm") {
			ethAddr := crypto.CreateAddress(keyAddr, 0)
			addr, _ := common.AddressParseFromBytes(ethAddr.Bytes())
			wingErc20 = common3.ConAddr{
				File:    file,
				Address: addr,
			}
			fmt.Println("wingErc20 token address:", addr.ToHexString())
			addrMap = append(addrMap, wingErc20)
		} else {
			if strings.HasSuffix(file, "bridge.avm") {
				bridge = common3.ConAddr{
					File:    file,
					Address: common.AddressFromVmCode(code),
				}
				fmt.Println("bridge address:", bridge.Address.ToHexString())
				addrMap = append(addrMap, bridge)
			} else if strings.HasSuffix(file, "WingToken.avm") {
				wingOep4 = common3.ConAddr{
					File:    file,
					Address: common.AddressFromVmCode(code),
				}
				fmt.Println("wingOep4 address:", wingOep4.Address.ToHexString())
				addrMap = append(addrMap, wingOep4)
			}
		}
		//addrMap = append(addrMap, conaddr)
	}

	// 调用 bridge init 方法
	admin, _ := common.AddressFromBase58("ARGK44mXXZfU6vcdSfFKMzjaabWxyog1qb")
	bridgeInit(admin, bridge, wingOep4, wingErc20, database, acct)

	// oep4 to erc20
	addr, _ := common.AddressParseFromBytes(keyAddr.Bytes())
	oep4ToErc20(admin, bridge, addr, database, acct, addrMap)

	return
	testContext := common3.TestContext{
		Admin:   admin,
		AddrMap: nil,
	}
	for _, item := range contract {
		file := item.File
		cont := item.Contract
		log.Infof("exacting testcase from %s", file)
		addr := common.AddressFromVmCode(cont)
		if strings.HasSuffix(file, ".avm") {
			testCases := GenNeoTextCaseTransaction(addr, database)
			for _, testCase := range testCases[0] { // only handle group 0 currently
				val, _ := json.Marshal(testCase)
				log.Info("executing testcase: ", string(val))
				tx, err := common3.GenNeoVMTransaction(testCase, addr, &testContext)
				checkErr(err)

				execTxCheckRes(tx, testCase, database, addr, acct)
			}
		} else if strings.HasSuffix(file, ".wasm") {
			testCases := ExactTestCase(cont)
			for _, testCase := range testCases[0] { // only handle group 0 currently
				val, _ := json.Marshal(testCase)
				log.Info("executing testcase: ", string(val))
				tx, err := common3.GenWasmTransaction(testCase, addr, &testContext)
				checkErr(err)

				execTxCheckRes(tx, testCase, database, addr, acct)
			}
		}
	}

	log.Info("contract test succeed")
}

type ExecEnv struct {
	Contract  common.Address
	Time      uint32
	Height    uint32
	Tx        *types.Transaction
	BlockHash common.Uint256
}

func checkExecResult(testCase common3.TestCase, result *states.PreExecResult, execEnv ExecEnv) {
	assertEq(result.State, byte(1))
	ret := result.Result.(string)
	switch testCase.Method {
	case "timestamp":
		sink := common.NewZeroCopySink(nil)
		sink.WriteUint64(uint64(execEnv.Time))
		assertEq(ret, hex.EncodeToString(sink.Bytes()))
	case "block_height":
		sink := common.NewZeroCopySink(nil)
		sink.WriteUint32(uint32(execEnv.Height))
		assertEq(ret, hex.EncodeToString(sink.Bytes()))
	case "self_address", "entry_address":
		assertEq(ret, hex.EncodeToString(execEnv.Contract[:]))
	case "caller_address":
		assertEq(ret, hex.EncodeToString(common.ADDRESS_EMPTY[:]))
	case "current_txhash":
		hash := execEnv.Tx.Hash()
		assertEq(ret, hex.EncodeToString(hash[:]))
	case "current_blockhash":
		assertEq(ret, hex.EncodeToString(execEnv.BlockHash[:]))
	//case "sha256":
	//	let data :&[u8]= source.read().unwrap();
	//	sink.write(runtime::sha256(&data))
	//}
	default:
		if len(testCase.Expect) != 0 {
			expect, err := utils.ParseParams(testCase.Expect)
			checkErr(err)
			if execEnv.Tx.TxType == types.InvokeNeo {
				val := buildNeoVmValueFromExpect(expect)
				cv, err := val.ConvertNeoVmValueHexString()
				checkErr(err)
				assertEq(cv, result.Result)
			} else if execEnv.Tx.TxType == types.InvokeWasm {
				exp, err := utils2.BuildWasmContractParam(expect)
				checkErr(err)
				assertEq(ret, hex.EncodeToString(exp))
			} else {
				panic("error tx type")
			}
		}
		if len(testCase.Notify) != 0 {
			js, _ := json.Marshal(result.Notify)
			assertEq(true, strings.Contains(string(js), testCase.Notify))
		}
	}
}

func oep4ToErc20(admin common.Address, bridge common3.ConAddr, ethAcct common.Address, database *ledger.Ledger, acct *account.Account, addrMap []common3.ConAddr) {
	param := fmt.Sprintf("[address:%s,address:%s,int:1000]", admin.ToBase58(), ethAcct.ToBase58())
	testContext := common3.TestContext{
		Admin:   admin,
		AddrMap: addrMap,
	}

	tc := common3.TestCase{
		Env: common3.TestEnv{
			Witness: []common.Address{admin},
		},
		NeedContext: true,
		Method:      "oep4ToErc202",
		Param:       param,
		Expect:      "bool:true",
	}
	tx, err := common3.GenNeoVMTransaction(tc, bridge.Address, &testContext)
	checkErr(err)

	execTxCheckRes(tx, tc, database, bridge.Address, acct)
}

func bridgeInit(admin common.Address, bridge, wingOep4, wingErc20 common3.ConAddr, database *ledger.Ledger, acct *account.Account) {
	//wing oep4 init
	param := "int:1"
	testContext := common3.TestContext{
		Admin:   admin,
		AddrMap: nil,
	}
	te := common3.TestEnv{Witness: []common.Address{admin, acct.Address}}
	tc := common3.NewTestCase(te, false, "init", param, "bool:true", "")
	tx, err := common3.GenNeoVMTransaction(tc, wingOep4.Address, &testContext)
	checkErr(err)
	execTxCheckRes(tx, tc, database, wingOep4.Address, acct)

	// oep4 balanceOf
	tc = common3.NewTestCase(common3.TestEnv{}, false, "balanceOf", fmt.Sprintf("[address:%s]", admin.ToBase58()), "int:1000000000000000", "")
	tx, err = common3.GenNeoVMTransaction(tc, wingOep4.Address, &testContext)
	checkErr(err)
	res, err := database.PreExecuteContract(tx)
	checkErr(err)
	data := res.Result.(string)
	d, _ := hex.DecodeString(data)
	if common.BigIntFromNeoBytes(d).Cmp(big.NewInt(1000000000000000)) != 0 {
		panic(common.BigIntFromNeoBytes(d).String())
	}

	// bridge init
	param = fmt.Sprintf("[address:%s,address:%s]", wingOep4.Address.ToBase58(), wingErc20.Address.ToBase58())
	tc = common3.NewTestCase(te, false, "init", param, "bool:true", "")
	tx, err = common3.GenNeoVMTransaction(tc, bridge.Address, &testContext)
	checkErr(err)
	execTxCheckRes(tx, tc, database, bridge.Address, acct)

	// bridge get_ont_address
	tc = common3.NewTestCase(te, false, "get_ont_address", "int:1", "address:"+wingOep4.Address.ToBase58(), "")
	tx, err = common3.GenNeoVMTransaction(tc, bridge.Address, &testContext)
	checkErr(err)
	execTxCheckRes(tx, tc, database, bridge.Address, acct)

	// wing erc20 totalSupply
	evmTx, err := GenEVMTx(common4.BytesToAddress(wingErc20.Address[:]), "totalSupply")
	checkErr(err)
	tx, err = types.TransactionFromEIP155(evmTx)
	checkErr(err)
	res, err = database.PreExecuteContract(tx)
	checkErr(err)
	r := res.Result.(*evm.ExecutionResult)
	log.Infof("execute: %v", JsonString(r))
	fmt.Println(big.NewInt(0).SetBytes(r.ReturnData).String())

	// wingErc20 balanceOf
	evmTx, err = GenEVMTx(common4.BytesToAddress(wingErc20.Address[:]), "balanceOf", testEthAddr)
	checkErr(err)
	tx, err = types.TransactionFromEIP155(evmTx)
	checkErr(err)
	res, err = database.PreExecuteContract(tx)
	checkErr(err)
	r = res.Result.(*evm.ExecutionResult)
	fmt.Println(big.NewInt(0).SetBytes(r.ReturnData).String())
}

func GenEVMTx(contractAddr common4.Address, method string, params ...interface{}) (*types2.Transaction, error) {
	chainId := big.NewInt(0)
	opts, err := bind.NewKeyedTransactorWithChainID(testPrivateKey, chainId)
	opts.GasPrice = big.NewInt(0)
	opts.Nonce = big.NewInt(0)

	checkErr(err)
	parsed, err := abi.JSON(strings.NewReader(WingABI))
	checkErr(err)
	input, err := parsed.Pack(method, params...)
	deployTx := types2.NewTransaction(opts.Nonce.Uint64(), contractAddr, opts.Value, opts.GasLimit, opts.GasPrice, input)
	signedTx, err := opts.Signer(opts.From, deployTx)
	checkErr(err)
	return signedTx, err
}

func buildNeoVmValueFromExpect(expectlist []interface{}) *vmtypes.VmValue {
	if len(expectlist) > 1 {
		panic("only support return one value")
	}
	expect := expectlist[0]

	switch expect.(type) {
	case string:
		val, err := vmtypes.VmValueFromBytes([]byte(expect.(string)))
		if err != nil {
			panic(err)
		}
		return &val
	case []byte:
		val, err := vmtypes.VmValueFromBytes(expect.([]byte))
		if err != nil {
			panic(err)
		}
		return &val
	case int64:
		val := vmtypes.VmValueFromInt64(expect.(int64))
		return &val
	case bool:
		val := vmtypes.VmValueFromBool(expect.(bool))
		return &val
	case common.Address:
		addr := expect.(common.Address)
		val, err := vmtypes.VmValueFromBytes(addr[:])
		if err != nil {
			panic(err)
		}
		return &val
	default:
		fmt.Printf("unspport param type %s", reflect.TypeOf(expect))
		panic("unspport param type")
	}
}

func GenAccounts(num int) []*account.Account {
	var accounts []*account.Account
	for i := 0; i < num; i++ {
		acc := account.NewAccount("")
		accounts = append(accounts, acc)
	}
	return accounts
}

func makeBlock(acc *account.Account, txs []*types.Transaction) (*types.Block, error) {
	nextBookkeeper, err := types.AddressFromBookkeepers([]keypair.PublicKey{acc.PublicKey})
	if err != nil {
		return nil, fmt.Errorf("GetBookkeeperAddress error:%s", err)
	}
	prevHash := ledger.DefLedger.GetCurrentBlockHash()
	height := ledger.DefLedger.GetCurrentBlockHeight()

	nonce := uint64(height)
	var txHash []common.Uint256
	for _, t := range txs {
		txHash = append(txHash, t.Hash())
	}

	txRoot := common.ComputeMerkleRoot(txHash)

	blockRoot := ledger.DefLedger.GetBlockRootWithNewTxRoots(height+1, []common.Uint256{txRoot})
	header := &types.Header{
		Version:          0,
		PrevBlockHash:    prevHash,
		TransactionsRoot: txRoot,
		BlockRoot:        blockRoot,
		Timestamp:        constants.GENESIS_BLOCK_TIMESTAMP + height + 1,
		Height:           height + 1,
		ConsensusData:    nonce,
		NextBookkeeper:   nextBookkeeper,
	}
	block := &types.Block{
		Header:       header,
		Transactions: txs,
	}

	blockHash := block.Hash()

	sig, err := signature.Sign(acc, blockHash[:])
	if err != nil {
		return nil, fmt.Errorf("signature, Sign error:%s", err)
	}

	block.Header.Bookkeepers = []keypair.PublicKey{acc.PublicKey}
	block.Header.SigData = [][]byte{sig}
	return block, nil
}

func assertEq(a interface{}, b interface{}) {
	if reflect.DeepEqual(a, b) == false {
		panic(fmt.Sprintf("not equal: a= %v, b=%v", a, b))
	}
}

func JsonString(v interface{}) string {
	buf, err := json.MarshalIndent(v, "", "  ")
	checkErr(err)

	return string(buf)
}
