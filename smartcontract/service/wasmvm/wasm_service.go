///*
// * Copyright (C) 2018 The ontology Authors
// * This file is part of The ontology library.
// *
// * The ontology is free software: you can redistribute it and/or modify
// * it under the terms of the GNU Lesser General Public License as published by
// * the Free Software Foundation, either version 3 of the License, or
// * (at your option) any later version.
// *
// * The ontology is distributed in the hope that it will be useful,
// * but WITHOUT ANY WARRANTY; without even the implied warranty of
// * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// * GNU Lesser General Public License for more details.
// *
// * You should have received a copy of the GNU Lesser General Public License
// * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
// */
package wasmvm

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/store"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/context"
	"github.com/ontio/ontology/smartcontract/event"
	"github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
	"github.com/ontio/ontology/smartcontract/states"
	"github.com/ontio/ontology/smartcontract/storage"
	"github.com/ontio/ontology/vm/wasmvm/exec"
	"github.com/ontio/ontology/vm/wasmvm/util"
)

type WasmVmService struct {
	Store   store.LedgerStore
	CacheDB *storage.CacheDB
	//CloneCache    *storage.CloneCache
	ContextRef    context.ContextRef
	Notifications []*event.NotifyEventInfo
	Code          []byte
	Tx            *types.Transaction
	Time          uint32
	Height        uint32
	RandomHash    common.Uint256
}

var (
	ERR_CHECK_STACK_SIZE  = errors.NewErr("[WasmVmService] vm over max stack size!")
	ERR_EXECUTE_CODE      = errors.NewErr("[WasmVmService] vm execute code invalid!")
	ERR_GAS_INSUFFICIENT  = errors.NewErr("[WasmVmService] gas insufficient")
	VM_EXEC_STEP_EXCEED   = errors.NewErr("[WasmVmService] vm execute step exceed!")
	CONTRACT_NOT_EXIST    = errors.NewErr("[WasmVmService] Get contract code from db fail")
	DEPLOYCODE_TYPE_ERROR = errors.NewErr("[WasmVmService] DeployCode type error!")
	VM_EXEC_FAULT         = errors.NewErr("[WasmVmService] vm execute state fault!")
)

func (this *WasmVmService) Invoke() (interface{}, error) {

	if len(this.Code) == 0 {
		return nil, ERR_EXECUTE_CODE
	}

	stateMachine := NewWasmStateMachine()
	//register the "CallContract" function
	stateMachine.Register("ONT_CallContract", this.callContract)

	stateMachine.Register("ONT_NativeInvoke", this.nativeInvoke)
	stateMachine.Register("ONT_MarshalNativeParams", this.marshalNativeParams)
	//stateMachine.Register("ONT_MarshalNeoParams", this.marshalNeoParams)
	//runtime
	stateMachine.Register("ONT_Runtime_CheckWitness", this.runtimeCheckWitness)
	stateMachine.Register("ONT_Runtime_Notify", this.runtimeNotify)
	stateMachine.Register("ONT_Runtime_CheckSig", this.runtimeCheckSig)
	stateMachine.Register("ONT_Runtime_GetTime", this.runtimeGetTime)
	stateMachine.Register("ONT_Runtime_Log", this.runtimeLog)
	//attribute
	stateMachine.Register("ONT_Attribute_GetUsage", this.attributeGetUsage)
	stateMachine.Register("ONT_Attribute_GetData", this.attributeGetData)
	//block
	stateMachine.Register("ONT_Block_GetCurrentHeaderHash", this.blockGetCurrentHeaderHash)
	stateMachine.Register("ONT_Block_GetCurrentHeaderHeight", this.blockGetCurrentHeaderHeight)
	stateMachine.Register("ONT_Block_GetCurrentBlockHash", this.blockGetCurrentBlockHash)
	stateMachine.Register("ONT_Block_GetCurrentBlockHeight", this.blockGetCurrentBlockHeight)
	stateMachine.Register("ONT_Block_GetTransactionByHash", this.blockGetTransactionByHash)
	stateMachine.Register("ONT_Block_GetTransactionCount", this.blockGetTransactionCount)
	stateMachine.Register("ONT_Block_GetTransactions", this.blockGetTransactions)

	//blockchain
	stateMachine.Register("ONT_BlockChain_GetHeight", this.blockChainGetHeight)
	stateMachine.Register("ONT_BlockChain_GetHeaderByHeight", this.blockChainGetHeaderByHeight)
	stateMachine.Register("ONT_BlockChain_GetHeaderByHash", this.blockChainGetHeaderByHash)
	stateMachine.Register("ONT_BlockChain_GetBlockByHeight", this.blockChainGetBlockByHeight)
	stateMachine.Register("ONT_BlockChain_GetBlockByHash", this.blockChainGetBlockByHash)
	stateMachine.Register("ONT_BlockChain_GetContract", this.blockChainGetContract)

	//header
	stateMachine.Register("ONT_Header_GetHash", this.headerGetHash)
	stateMachine.Register("ONT_Header_GetVersion", this.headerGetVersion)
	stateMachine.Register("ONT_Header_GetPrevHash", this.headerGetPrevHash)
	stateMachine.Register("ONT_Header_GetMerkleRoot", this.headerGetMerkleRoot)
	stateMachine.Register("ONT_Header_GetIndex", this.headerGetIndex)
	stateMachine.Register("ONT_Header_GetTimestamp", this.headerGetTimestamp)
	stateMachine.Register("ONT_Header_GetConsensusData", this.headerGetConsensusData)
	stateMachine.Register("ONT_Header_GetNextConsensus", this.headerGetNextConsensus)

	//storage
	stateMachine.Register("ONT_Storage_Put", this.putstore)
	stateMachine.Register("ONT_Storage_Get", this.getstore)
	stateMachine.Register("ONT_Storage_Delete", this.deletestore)

	//transaction
	stateMachine.Register("ONT_Transaction_GetHash", this.transactionGetHash)
	stateMachine.Register("ONT_Transaction_GetType", this.transactionGetType)
	stateMachine.Register("ONT_Transaction_GetAttributes", this.transactionGetAttributes)

	engine := exec.NewExecutionEngine(
		new(util.ECDsaCrypto),
		stateMachine,
	)

	contract := &states.ContractInvokeParam{}
	contract.Deserialize(bytes.NewBuffer(this.Code))
	//addr := contract.Address
	fmt.Printf("contract address is %s\n", contract.Address.ToBase58())

	code, err := this.Store.GetContractState(contract.Address)
	if err != nil{
		fmt.Printf("getContract error:%s\n",err)
		return nil,err
	}


	this.ContextRef.PushContext(&context.Context{ContractAddress:contract.Address, Code: code.Code})

	//if contract.Code == nil {
	//	dpcode, err := this.GetContractCodeFromAddress(addr)
	//	if err != nil {
	//		return nil, errors.NewErr("get contract  error")
	//	}
	//	contract.Code = dpcode
	//}

	var caller common.Address
	if this.ContextRef.CallingContext() == nil {
		caller = common.Address{}
	} else {
		caller = this.ContextRef.CallingContext().ContractAddress
	}
	this.ContextRef.PushContext(&context.Context{ContractAddress: contract.Address})
	fmt.Printf("===this.Code:%v\n",code.Code)
	fmt.Printf("===contract.Method:%v\n",contract.Method)
	fmt.Printf("===contract.Args:%v\n",contract.Args)
	fmt.Printf("===contract.Version:%v\n",contract.Version)


	res, err := engine.Call(caller, code.Code, contract.Method, contract.Args, contract.Version)

	if err != nil {
		fmt.Printf("wasm call error:%s\n",err)

		return nil, err
	}

	//get the return message
	result, err := engine.GetVM().GetPointerMemory(uint64(binary.LittleEndian.Uint32(res)))
	if err != nil {
		fmt.Printf("wasm call GetPointerMemory error:%s\n",err)

		return nil, err
	}

	this.ContextRef.PopContext()
	this.ContextRef.PushNotifications(this.Notifications)
	return result, nil
}

//
//func (this *WasmVmService) marshalNeoParams(engine *exec.ExecutionEngine) (bool, error) {
//	vm := engine.GetVM()
//	envCall := vm.GetEnvCall()
//	params := envCall.GetParams()
//	if len(params) != 1 {
//		return false, errors.NewErr("[marshalNeoParams]parameter count error while call marshalNativeParams")
//	}
//	argbytes, err := vm.GetPointerMemory(params[0])
//	if err != nil {
//		return false, err
//	}
//	bytesLen := len(argbytes)
//	args := make([]interface{}, bytesLen/8)
//	icount := 0
//	for i := 0; i < bytesLen; i += 8 {
//		tmpBytes := argbytes[i : i+8]
//		ptype, err := vm.GetPointerMemory(uint64(binary.LittleEndian.Uint32(tmpBytes[:4])))
//		if err != nil {
//			return false, err
//		}
//		pvalue, err := vm.GetPointerMemory(uint64(binary.LittleEndian.Uint32(tmpBytes[4:8])))
//		if err != nil {
//			return false, err
//		}
//		switch strings.ToLower(util.TrimBuffToString(ptype)) {
//		case "string":
//			args[icount] = util.TrimBuffToString(pvalue)
//		case "int":
//			args[icount], err = strconv.Atoi(util.TrimBuffToString(pvalue))
//			if err != nil {
//				return false, err
//			}
//		case "int64":
//			args[icount], err = strconv.ParseInt(util.TrimBuffToString(pvalue), 10, 64)
//			if err != nil {
//				return false, err
//			}
//		default:
//			args[icount] = util.TrimBuffToString(pvalue)
//		}
//		icount++
//	}
//	builder := neovm.NewParamsBuilder(bytes.NewBuffer(nil))
//	err = buildNeoVMParamInter(builder, []interface{}{args})
//	if err != nil {
//		return false, err
//	}
//	neoargs := builder.ToArray()
//	idx, err := vm.SetPointerMemory(neoargs)
//	if err != nil {
//		return false, err
//	}
//	vm.RestoreCtx()
//	vm.PushResult(uint64(idx))
//	return true, nil
//
//}

// marshalNativeParams
// make parameter bytes for call native contract
func (this *WasmVmService) marshalNativeParams(engine *exec.ExecutionEngine) (bool, error) {
	vm := engine.GetVM()
	envCall := vm.GetEnvCall()
	params := envCall.GetParams()
	if len(params) != 1 {
		return false, errors.NewErr("[callContract]parameter count error while call marshalNativeParams")
	}

	transferbytes, err := vm.GetPointerMemory(params[0])
	if err != nil {
		return false, err
	}
	//transferbytes is a nested struct with states.Transfer
	//type Transfers struct {
	//	States  []*State		   -------->i32 pointer 4 bytes
	//}
	if len(transferbytes) != 4 {
		return false, errors.NewErr("[callContract]parameter format error while call marshalNativeParams")
	}
	transfer := &ont.Transfers{}

	statesAddr := binary.LittleEndian.Uint32(transferbytes[:4])
	statesbytes, err := vm.GetPointerMemory(uint64(statesAddr))
	if err != nil {
		return false, err
	}

	//statesbytes is slice of struct with states.
	//type State struct {
	//	From    common.Address  -------->i32 pointer 4 bytes
	//	To      common.Address  -------->i32 pointer 4 bytes
	//	Value   *big.Int        -------->i64 8 bytes
	//}
	//total is 4 + 4 + 8 = 24 bytes
	statecnt := len(statesbytes) / 16
	states := make([]ont.State, statecnt)

	for i := 0; i < statecnt; i++ {
		tmpbytes := statesbytes[i*16 : (i+1)*16]
		state := ont.State{}
		fromAddessBytes, err := vm.GetPointerMemory(uint64(binary.LittleEndian.Uint32(tmpbytes[:4])))
		if err != nil {
			return false, err
		}
		fromAddress, err := common.AddressFromBase58(util.TrimBuffToString(fromAddessBytes))
		if err != nil {
			return false, err
		}
		state.From = fromAddress

		toAddressBytes, err := vm.GetPointerMemory(uint64(binary.LittleEndian.Uint32(tmpbytes[4:8])))
		if err != nil {
			return false, err
		}
		toAddress, err := common.AddressFromBase58(util.TrimBuffToString(toAddressBytes))
		state.To = toAddress
		//tmpbytes[12:16] is padding
		amount := binary.LittleEndian.Uint64(tmpbytes[8:])
		state.Value = amount
		states[i] = state

	}

	transfer.States = states
	tbytes := new(bytes.Buffer)
	transfer.Serialize(tbytes)

	result, err := vm.SetPointerMemory(tbytes.Bytes())
	if err != nil {
		return false, err
	}
	vm.RestoreCtx()
	vm.PushResult(uint64(result))
	return true, nil
}

//get contract code from address
func (this *WasmVmService) getContract(bs []byte) ([]byte, error) {
	//item, err := this.CacheDB.GetContract(address)
	//if err != nil {
	//	return nil, errors.NewErr("[getContract] Get contract context error!")
	//}
	//log.Debugf("invoke contract address:%x", common.ToArrayReverse(address))
	//if item == nil {
	//	return nil, CONTRACT_NOT_EXIST
	//}
	//contract, ok := item.Value.(*payload.DeployCode)
	//if !ok {
	//	return nil, DEPLOYCODE_TYPE_ERROR
	//}
	//return contract.Code, nil
	address, err := common.AddressParseFromBytes(bs)
	dep, err := this.CacheDB.GetContract(address)
	if err != nil {
		return nil, errors.NewErr("[getContract] Get contract context error!")
	}
	log.Debugf("invoke contract address:%s", address.ToHexString())
	if dep == nil {
		return nil, CONTRACT_NOT_EXIST
	}
	return dep.Code, nil
}

//native invoke
func (this *WasmVmService) nativeInvoke(engine *exec.ExecutionEngine) (bool, error) {
	vm := engine.GetVM()
	envCall := vm.GetEnvCall()
	params := envCall.GetParams()
	if len(params) != 4 {
		return false, errors.NewErr("[nativeInvoke]parameter count error while call readMessage")
	}
	//get version
	version := params[0]

	//get contractAddress
	contractAddressIdx := params[1]
	addr, err := vm.GetPointerMemory(contractAddressIdx)
	if err != nil {
		return false, errors.NewErr("[nativeInvoke]get Contract address failed:" + err.Error())
	}

	if len(addr) == 0 {
		return false, errors.NewErr("[nativeInvoke]No native contract address found!")
	}
	addrbytes, err := common.HexToBytes(util.TrimBuffToString(addr))

	contractAddress, err := common.AddressParseFromBytes(addrbytes)

	//get method
	methodIdx := params[2]
	method, err := vm.GetPointerMemory(methodIdx)
	if err != nil {
		return false, errors.NewErr("[nativeInvoke]get method failed!")
	}
	methodName := util.TrimBuffToString(method)

	//get args
	argsIdx := params[3]
	argsbytes, err := vm.GetPointerMemory(argsIdx)

	contract := states.ContractInvokeParam{
		Version: byte(version),
		Address: contractAddress,
		Method:  methodName,
		Args:    argsbytes,
	}

	sink := common.ZeroCopySink{}
	contract.Serialization(&sink)

	native := &native.NativeService{
		CacheDB:     this.CacheDB,
		InvokeParam: contract,
		Tx:          this.Tx,
		Height:      this.Height,
		Time:        this.Time,
		ContextRef:  this.ContextRef,
		ServiceMap:  make(map[string]native.Handler),
	}

	result, err := native.Invoke()
	if err != nil {
		return false, err
	}

	if err != nil {
		return false, errors.NewErr("[nativeInvoke]AppCall failed:" + err.Error())
	}

	this.ContextRef.PopContext()
	vm.RestoreCtx()
	//var res string
	if envCall.GetReturns() {
		//res = fmt.Sprintf("%s", result)

		idx, err := vm.SetPointerMemory(result)
		if err != nil {
			return false, errors.NewErr("[callContract]SetPointerMemory failed:" + err.Error())
		}
		vm.PushResult(uint64(idx))
	}

	return true, nil

}

// callContract
// need 4 parameters
//0: contract address
//1: contract code
//2: method name
//3: args
func (this *WasmVmService) callContract(engine *exec.ExecutionEngine) (bool, error) {
	vm := engine.GetVM()
	envCall := vm.GetEnvCall()
	params := envCall.GetParams()
	if len(params) != 4 {
		return false, errors.NewErr("[callContract]parameter count error while call readMessage")
	}
	var contractAddress common.Address
	//var contractBytes []byte
	//get contract address
	contractAddressIdx := params[0]
	addr, err := vm.GetPointerMemory(contractAddressIdx)
	if err != nil {
		return false, errors.NewErr("[callContract]get Contract address failed:" + err.Error())
	}

	if addr != nil {
		addrbytes, err := common.HexToBytes(util.TrimBuffToString(addr))
		if err != nil {
			return false, errors.NewErr("[callContract]get contract address error:" + err.Error())
		}
		contractAddress, err = common.AddressParseFromBytes(addrbytes)
		if err != nil {
			return false, errors.NewErr("[callContract]get contract address error:" + err.Error())
		}

	}

	//get contract code
	codeIdx := params[1]

	offchainContractCode, err := vm.GetPointerMemory(codeIdx)
	if err != nil {
		return false, errors.NewErr("[callContract]get Contract address failed:" + err.Error())
	}
	if offchainContractCode != nil {
		//contractBytes, err = common.HexToBytes(util.TrimBuffToString(offchainContractCode))
		//if err != nil {
		//	return false, err
		//
		//}
		//compute the offchain code address
		codestring := util.TrimBuffToString(offchainContractCode)
		//contractAddress = GetContractAddress(codestring, vmtypes.WASMVM)
		contractAddress, err = GetContractAddress(codestring)
	}
	fmt.Printf("")
	//get method
	methodName, err := vm.GetPointerMemory(params[2])
	if err != nil {
		return false, errors.NewErr("[callContract]get Contract methodName failed:" + err.Error())
	}
	//get args
	args, err := vm.GetPointerMemory(params[3])

	if err != nil {
		return false, errors.NewErr("[callContract]get Contract arg failed:" + err.Error())
	}
	this.ContextRef.PushContext(&context.Context{
		Code:            vm.VMCode,
		ContractAddress: vm.ContractAddress})
	//result, err := this.ContextRef.AppCall(contractAddress, util.TrimBuffToString(methodName), contractBytes, arg)

	bf := new(bytes.Buffer)
	contract := states.ContractInvokeParam{
		Version: 1, //fix to > 0
		Address: contractAddress,
		Method:  string(methodName),
		Args:    args,
	}

	if err := contract.Serialize(bf); err != nil {
		return false, err
	}

	this.Code = bf.Bytes()
	result, err := this.Invoke()
	if err != nil {
		return false, errors.NewErr("[callContract]AppCall failed:" + err.Error())
	}
	this.ContextRef.PopContext()
	vm.RestoreCtx()
	//var res string
	if envCall.GetReturns() {
		//res = fmt.Sprintf("%s", result)

		idx, err := vm.SetPointerMemory(result)
		if err != nil {
			return false, errors.NewErr("[callContract]SetPointerMemory failed:" + err.Error())
		}
		vm.PushResult(uint64(idx))
	}

	return true, nil
}

//func (this *WasmVmService) GetContractCodeFromAddress(address common.Address) ([]byte, error) {
//
//	dcode, err := this.Store.GetContractState(address)
//	if err != nil {
//		return nil, err
//	}
//
//	if dcode == nil {
//		return nil, errors.NewErr("[GetContractCodeFromAddress] deployed code is nil")
//	}
//
//	return dcode.Code.Code, nil
//
//}

//func (this *WasmVmService) getContractFromAddr(addr []byte) ([]byte, error) {
//	addrbytes, err := common.HexToBytes(util.TrimBuffToString(addr))
//	if err != nil {
//		return nil, errors.NewErr("get contract address error")
//	}
//	contactaddress, err := common.AddressParseFromBytes(addrbytes)
//	if err != nil {
//		return nil, errors.NewErr("get contract address error")
//	}
//	dpcode, err := this.GetContractCodeFromAddress(contactaddress)
//	if err != nil {
//		return nil, errors.NewErr("get contract  error")
//	}
//	return dpcode, nil
//}

//GetContractAddress return contract address
func GetContractAddress(code string) (common.Address, error) {
	data, err := hex.DecodeString(code)
	if err != nil {
		return common.Address{}, err
	}

	return common.AddressFromVmCode(data), nil
}

//will not call NEO contract
//buildNeoVMParamInter build neovm invoke param code
//func buildNeoVMParamInter(builder *neovm.ParamsBuilder, smartContractParams []interface{}) error {
//	//VM load params in reverse order
//	for i := len(smartContractParams) - 1; i >= 0; i-- {
//		switch v := smartContractParams[i].(type) {
//		case bool:
//			builder.EmitPushBool(v)
//		case int:
//			builder.EmitPushInteger(big.NewInt(int64(v)))
//		case uint:
//			builder.EmitPushInteger(big.NewInt(int64(v)))
//		case int32:
//			builder.EmitPushInteger(big.NewInt(int64(v)))
//		case uint32:
//			builder.EmitPushInteger(big.NewInt(int64(v)))
//		case int64:
//			builder.EmitPushInteger(big.NewInt(int64(v)))
//		case common.Fixed64:
//			builder.EmitPushInteger(big.NewInt(int64(v.GetData())))
//		case uint64:
//			val := big.NewInt(0)
//			builder.EmitPushInteger(val.SetUint64(uint64(v)))
//		case string:
//			builder.EmitPushByteArray([]byte(v))
//		case *big.Int:
//			builder.EmitPushInteger(v)
//		case []byte:
//			builder.EmitPushByteArray(v)
//		case []interface{}:
//			err := buildNeoVMParamInter(builder, v)
//			if err != nil {
//				return err
//			}
//			builder.EmitPushInteger(big.NewInt(int64(len(v))))
//			builder.Emit(neovm.PACK)
//		default:
//			return fmt.Errorf("unsupported param:%s", v)
//		}
//	}
//	return nil
//}
