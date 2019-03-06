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
package wasmvm

import (
	"bytes"
	"encoding/gob"
	"reflect"

	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/common/serialization"
	"github.com/ontio/ontology/core/payload"
	states2 "github.com/ontio/ontology/core/states"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/context"
	"github.com/ontio/ontology/smartcontract/event"
	native2 "github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
	"github.com/ontio/ontology/smartcontract/states"
	neotypes "github.com/ontio/ontology/vm/neovm/types"
)

type ContractType byte

const (
	NATIVE_CONTRACT ContractType = iota
	NEOVM_CONTRACT
	WASMVM_CONTRACT
	UNKOWN_CONTRACT
)

type Runtime struct {
	Service    *WasmVmService
	Input      []byte
	Output     []byte
	CallOutPut []byte
}

func (self *Runtime) TimeStamp(proc *exec.Process) uint64 {
	self.checkGas(TIME_STAMP_GAS)
	return uint64(self.Service.Time)
}

func (self *Runtime) BlockHeight(proc *exec.Process) uint32 {
	self.checkGas(BLOCK_HEGHT_GAS)
	return self.Service.Height
}

func (self *Runtime) SelfAddress(proc *exec.Process, dst uint32) {
	self.checkGas(SELF_ADDRESS_GAS)
	selfaddr := self.Service.ContextRef.CurrentContext().ContractAddress
	_, err := proc.WriteAt(selfaddr[:], int64(dst))
	if err != nil {
		panic(err)
	}
}

func (self *Runtime) CallerAddress(proc *exec.Process, dst uint32) {
	self.checkGas(CALLER_ADDRESS_GAS)
	if self.Service.ContextRef.CallingContext() != nil {
		calleraddr := self.Service.ContextRef.CallingContext().ContractAddress
		_, err := proc.WriteAt(calleraddr[:], int64(dst))
		if err != nil {
			panic(err)
		}
	} else {
		_, err := proc.WriteAt(common.ADDRESS_EMPTY[:], int64(dst))
		if err != nil {
			panic(err)
		}
	}

}

func (self *Runtime) EntryAddress(proc *exec.Process, dst uint32) {
	self.checkGas(ENTRY_ADDRESS_GAS)
	entryAddress := self.Service.ContextRef.EntryContext().ContractAddress
	_, err := proc.WriteAt(entryAddress[:], int64(dst))
	if err != nil {
		panic(err)
	}
}

func (self *Runtime) Checkwitness(proc *exec.Process, dst uint32) uint32 {
	self.checkGas(CHECKWITNESS_GAS)
	addrbytes := make([]byte, 20)
	_, err := proc.ReadAt(addrbytes, int64(dst))
	if err != nil {
		panic(err)
	}

	address, err := common.AddressParseFromBytes(addrbytes)
	if err != nil {
		panic(err)
	}

	if self.Service.ContextRef.CheckWitness(address) {
		return 1
	}
	return 0
}

func (self *Runtime) Ret(proc *exec.Process, ptr uint32, len uint32) {
	bs := make([]byte, len)
	_, err := proc.ReadAt(bs, int64(ptr))
	if err != nil {
		panic(err)
	}

	self.Output = make([]byte, len)
	copy(self.Output, bs)

	panic(nil)
}

func (self *Runtime) Debug(proc *exec.Process, ptr uint32, len uint32) {
	bs := make([]byte, len)
	_, err := proc.ReadAt(bs, int64(ptr))
	if err != nil {
		panic(err)
	}

	log.Debugf("[WasmContract Debug log]:%v\n", bs)
}

func (self *Runtime) Notify(proc *exec.Process, ptr uint32, len uint32) {
	bs := make([]byte, len)
	_, err := proc.ReadAt(bs, int64(ptr))
	if err != nil {
		panic(err)
	}

	notify := &event.NotifyEventInfo{self.Service.ContextRef.CurrentContext().ContractAddress, bs}
	notifys := make([]*event.NotifyEventInfo, 1)
	notifys[0] = notify
	self.Service.ContextRef.PushNotifications(notifys)
}

func (self *Runtime) InputLength(proc *exec.Process) uint32 {
	return uint32(len(self.Input))
}

func (self *Runtime) GetInput(proc *exec.Process, dst uint32) {
	_, err := proc.WriteAt(self.Input, int64(dst))
	if err != nil {
		panic(err)
	}
}

func (self *Runtime) CallOutputLength(proc *exec.Process) uint32 {
	return uint32(len(self.CallOutPut))
}

func (self *Runtime) GetCallOut(proc *exec.Process, dst uint32) {
	_, err := proc.WriteAt(self.CallOutPut, int64(dst))
	if err != nil {
		panic(err)
	}
}

func (self *Runtime) CallContract(proc *exec.Process, contractAddr uint32, inputPtr uint32, inputLen uint32) uint32 {
	contractAddrbytes := make([]byte, 20)
	_, err := proc.ReadAt(contractAddrbytes, int64(contractAddr))
	if err != nil {
		panic(err)
	}

	contractAddress, err := common.AddressParseFromBytes(contractAddrbytes)
	if err != nil {
		panic(err)
	}

	inputs := make([]byte, inputLen)
	_, err = proc.ReadAt(inputs, int64(inputPtr))
	if err != nil {
		panic(err)
	}

	contracttype, err := self.getContractType(contractAddress)
	if err != nil {
		panic(err)
	}

	currentCtx := &context.Context{
		Code:            self.Service.Code,
		ContractAddress: self.Service.ContextRef.CurrentContext().ContractAddress,
	}
	self.Service.ContextRef.PushContext(currentCtx)

	var result interface{}

	switch contracttype {
	case NATIVE_CONTRACT:
		bf := bytes.NewBuffer(inputs)
		ver, err := serialization.ReadByte(bf)
		if err != nil {
			panic(err)
		}

		method, err := serialization.ReadString(bf)
		if err != nil {
			panic(err)
		}

		args, err := serialization.ReadVarBytes(bf)
		if err != nil {
			panic(err)
		}

		contract := states.ContractInvokeParam{
			Version: ver,
			Address: contractAddress,
			Method:  method,
			Args:    args,
		}

		self.checkGas(NATIVE_INVOKE_GAS)
		native := &native2.NativeService{
			CacheDB:     self.Service.CacheDB,
			InvokeParam: contract,
			Tx:          self.Service.Tx,
			Height:      self.Service.Height,
			Time:        self.Service.Time,
			ContextRef:  self.Service.ContextRef,
			ServiceMap:  make(map[string]native2.Handler),
		}

		result, err = native.Invoke()
		if err != nil {
			panic(errors.NewErr("[nativeInvoke]AppCall failed:" + err.Error()))
		}

	case WASMVM_CONTRACT:
		//contract
		self.checkGas(CALL_CONTRACT_GAS)
		bf := bytes.NewBuffer(nil)
		//if err := contract.Serialize(bf); err != nil {
		//	panic(err)
		//}
		conParam := states.WasmContractParam{Address: contractAddress, Args: inputs}
		if err := conParam.Serialize(bf); err != nil {
			panic(err)
		}

		newservice, err := self.Service.ContextRef.NewExecuteEngine(bf.Bytes(), types.InvokeWasm)
		if err != nil {
			panic(err)
		}

		result, err = newservice.Invoke()
		if err != nil {
			panic(err)
		}

	case NEOVM_CONTRACT:
		self.checkGas(CALL_CONTRACT_GAS)
		neoservice, err := self.Service.ContextRef.NewExecuteEngine(inputs, types.InvokeNeo)
		if err != nil {
			panic(err)
		}
		tmp, err := neoservice.Invoke()
		if err != nil {
			panic(err)
		}
		switch tmp.(type) {
		case neotypes.StackItems:
			result, err = tmp.(neotypes.StackItems).GetByteArray()
			if err != nil {
				panic(err)
			}
		default:
			result = tmp
		}

	default:
		panic(errors.NewErr("Not a supported contract type"))
	}

	self.Service.ContextRef.PopContext()

	buf := bytes.NewBuffer(nil)

	enc := gob.NewEncoder(buf)
	err = enc.Encode(result)
	if err != nil {
		panic(errors.NewErr("[callContract]callContract failed:" + err.Error()))
	}

	bs := buf.Bytes()
	self.CallOutPut = make([]byte, len(bs))

	copy(self.CallOutPut, bs)
	return uint32(len(self.CallOutPut))
}

func NewHostModule(host *Runtime) *wasm.Module {
	m := wasm.NewModule()
	m.Types = &wasm.SectionTypes{
		Entries: []wasm.FunctionSig{
			//func()uint64    [0]
			{
				Form:        0, // value for the 'func' type constructor
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI64},
			},
			//func()uint32     [1]
			{
				Form:        0, // value for the 'func' type constructor
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			//func(uint32)     [2]
			{
				Form:       0, // value for the 'func' type constructor
				ParamTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			//func(uint32)uint32  [3]
			{
				Form:        0, // value for the 'func' type constructor
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			//func(uint32,uint32)  [4]
			{
				Form:       0, // value for the 'func' type constructor
				ParamTypes: []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32},
			},
			//func(uint32,uint32,uint32)uint32  [5]
			{
				Form:        0, // value for the 'func' type constructor
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			//func(uint32,uint32,uint32,uint32,uint32)uint32  [6]
			{
				Form:        0, // value for the 'func' type constructor
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			//func(uint32,uint32,uint32,uint32)  [7]
			{
				Form:       0, // value for the 'func' type constructor
				ParamTypes: []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32},
			},
			//func(uint32,uint32)uint32   [8]
			{
				Form:        0, // value for the 'func' type constructor
				ParamTypes:  []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			//func(uint32 * 12)uint32   [9]
			{
				Form: 0, // value for the 'func' type constructor
				ParamTypes: []wasm.ValueType{wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32,
					wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32,
					wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32,
					wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32, wasm.ValueTypeI32,
					wasm.ValueTypeI32, wasm.ValueTypeI32},
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			//funct()   [10]
			{
				Form: 0, // value for the 'func' type constructor
			},
		},
	}
	m.FunctionIndexSpace = []wasm.Function{
		{
			Sig:  &m.Types.Entries[0],
			Host: reflect.ValueOf(host.TimeStamp),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[1],
			Host: reflect.ValueOf(host.BlockHeight),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[1],
			Host: reflect.ValueOf(host.InputLength),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[1],
			Host: reflect.ValueOf(host.CallOutputLength),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[2],
			Host: reflect.ValueOf(host.SelfAddress),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[2],
			Host: reflect.ValueOf(host.CallerAddress),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[2],
			Host: reflect.ValueOf(host.EntryAddress),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[2],
			Host: reflect.ValueOf(host.GetInput),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[2],
			Host: reflect.ValueOf(host.GetCallOut),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[3],
			Host: reflect.ValueOf(host.Checkwitness),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[3],
			Host: reflect.ValueOf(host.GetCurrentBlockHash),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[3],
			Host: reflect.ValueOf(host.GetCurrentTxHash),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[4],
			Host: reflect.ValueOf(host.Ret),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[4],
			Host: reflect.ValueOf(host.Notify),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[4],
			Host: reflect.ValueOf(host.Debug),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[5],
			Host: reflect.ValueOf(host.CallContract),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[6],
			Host: reflect.ValueOf(host.StorageRead),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[7],
			Host: reflect.ValueOf(host.StorageWrite),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[8],
			Host: reflect.ValueOf(host.StorageDelete),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[9],
			Host: reflect.ValueOf(host.ContractCreate),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[9],
			Host: reflect.ValueOf(host.ContractMigrate),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
		{
			Sig:  &m.Types.Entries[10],
			Host: reflect.ValueOf(host.ContractDelete),
			Body: &wasm.FunctionBody{}, // create a dummy wasm body (the actual value will be taken from Host.)
		},
	}

	m.Export = &wasm.SectionExports{
		Entries: map[string]wasm.ExportEntry{
			"timestamp": {
				FieldStr: "timestamp",
				Kind:     wasm.ExternalFunction,
				Index:    0,
			},
			"block_height": {
				FieldStr: "block_height",
				Kind:     wasm.ExternalFunction,
				Index:    1,
			},
			"input_length": {
				FieldStr: "input_length",
				Kind:     wasm.ExternalFunction,
				Index:    2,
			},
			"call_output_length": {
				FieldStr: "call_output_length",
				Kind:     wasm.ExternalFunction,
				Index:    3,
			},
			"self_address": {
				FieldStr: "self_address",
				Kind:     wasm.ExternalFunction,
				Index:    4,
			},
			"caller_address": {
				FieldStr: "caller_address",
				Kind:     wasm.ExternalFunction,
				Index:    5,
			},
			"entry_address": {
				FieldStr: "entry_address",
				Kind:     wasm.ExternalFunction,
				Index:    6,
			},
			"get_input": {
				FieldStr: "get_input",
				Kind:     wasm.ExternalFunction,
				Index:    7,
			},
			"get_output": {
				FieldStr: "get_output",
				Kind:     wasm.ExternalFunction,
				Index:    8,
			},
			"check_witness": {
				FieldStr: "check_witness",
				Kind:     wasm.ExternalFunction,
				Index:    9,
			},
			"current_blockhash": {
				FieldStr: "get_current_blockhash",
				Kind:     wasm.ExternalFunction,
				Index:    10,
			},
			"current_txhash": {
				FieldStr: "get_current_txhash",
				Kind:     wasm.ExternalFunction,
				Index:    11,
			},
			"ret": {
				FieldStr: "ret",
				Kind:     wasm.ExternalFunction,
				Index:    12,
			},
			"notify": {
				FieldStr: "notify",
				Kind:     wasm.ExternalFunction,
				Index:    13,
			},
			"contract_debug": {
				FieldStr: "contract_debug",
				Kind:     wasm.ExternalFunction,
				Index:    14,
			},
			"call_contract": {
				FieldStr: "call_contract",
				Kind:     wasm.ExternalFunction,
				Index:    15,
			},
			"storage_read": {
				FieldStr: "storage_read",
				Kind:     wasm.ExternalFunction,
				Index:    16,
			},
			"storage_write": {
				FieldStr: "storage_write",
				Kind:     wasm.ExternalFunction,
				Index:    17,
			},
			"contract_create": {
				FieldStr: "contract_create",
				Kind:     wasm.ExternalFunction,
				Index:    18,
			},
			"contract_migrate": {
				FieldStr: "contract_migrate",
				Kind:     wasm.ExternalFunction,
				Index:    19,
			},
			"contract_delete": {
				FieldStr: "contract_delete",
				Kind:     wasm.ExternalFunction,
				Index:    20,
			},
		},
	}

	return m
}

func (self *Runtime) getContractType(addr common.Address) (ContractType, error) {
	if utils.IsNativeContract(addr) {
		return NATIVE_CONTRACT, nil
	}

	dep, err := self.Service.CacheDB.GetContract(addr)
	if err != nil {
		return UNKOWN_CONTRACT, err
	}
	if dep == nil {
		return UNKOWN_CONTRACT, errors.NewErr("contract is not exist.")
	}
	if dep.VmType == payload.WASMVM_TYPE {
		return WASMVM_CONTRACT, nil
	}

	return NEOVM_CONTRACT, nil

}

func (self *Runtime) checkGas(gaslimit uint64) {
	gas := self.Service.vm.AvaliableGas
	if gas.GasLimit >= gaslimit {
		gas.GasLimit -= gaslimit
	} else {
		panic(errors.NewErr("[wasm_Service]Insufficient gas limit"))
	}
}

func serializeStorageKey(contractAddress common.Address, key []byte) ([]byte, error) {
	bf := new(bytes.Buffer)
	storageKey := &states2.StorageKey{ContractAddress: contractAddress, Key: key}
	if _, err := storageKey.Serialize(bf); err != nil {
		return []byte{}, errors.NewErr("[serializeStorageKey] StorageKey serialize error!")
	}
	return bf.Bytes(), nil
}
