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
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/serialization"
	states2 "github.com/ontio/ontology/core/states"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/context"
	"github.com/ontio/ontology/smartcontract/event"
	native2 "github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
	"github.com/ontio/ontology/smartcontract/states"
	"reflect"
	"github.com/ontio/ontology/core/types"
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
	return uint64(self.Service.Time)
}

func (self *Runtime) BlockHeight(proc *exec.Process) uint32 {
	return self.Service.Height
}

func (self *Runtime) SelfAddress(proc *exec.Process, dst uint32) {
	selfaddr := self.Service.ContextRef.CurrentContext().ContractAddress
	_, err := proc.WriteAt(selfaddr[:], int64(dst))
	if err != nil {
		panic(err)
	}
}

func (self *Runtime) CallerAddress(proc *exec.Process, dst uint32) {
	calleraddr := self.Service.ContextRef.CallingContext().ContractAddress
	_, err := proc.WriteAt(calleraddr[:], int64(dst))
	if err != nil {
		panic(err)
	}
}

func (self *Runtime) EntryAddress(proc *exec.Process, dst uint32) {
	entryAddress := self.Service.ContextRef.EntryContext().ContractAddress
	_, err := proc.WriteAt(entryAddress[:], int64(dst))
	if err != nil {
		panic(err)
	}
}

func (self *Runtime) Checkwitness(proc *exec.Process, dst uint32) uint32 {
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

	bf := bytes.NewBuffer(inputs)
	ver, err := serialization.ReadUint32(bf)
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

	contract := states.ContractInvokeParam{
		Version: byte(ver),
		Address: contractAddress,
		Method:  method,
		Args:    args,
	}

	switch contracttype {
	case NATIVE_CONTRACT:

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
		bf := bytes.NewBuffer(nil)
		if err := contract.Serialize(bf); err != nil {
			panic(err)
		}

		newservice, err := self.Service.ContextRef.NewExecuteEngine(bf.Bytes(),types.InvokeWasm)
		if err != nil {
			panic(err)
		}
		result, err = newservice.Invoke()
		if err != nil {
			panic(err)
		}

	case NEOVM_CONTRACT:
		//todo test if this work for neovm
		bf := bytes.NewBuffer(nil)
		if err := contract.Serialize(bf); err != nil {
			panic(err)
		}

		neoservice, err := self.Service.ContextRef.NewExecuteEngine(bf.Bytes(),types.Invoke)
		if err != nil {
			panic(err)
		}
		result, err = neoservice.Invoke()
		if err != nil {
			panic(err)
		}
		//new neovm_service

	default:
		panic(errors.NewErr("Not a supported contract type"))
	}
	self.Service.ContextRef.PopContext()

	buf := bytes.NewBuffer(nil)

	enc := gob.NewEncoder(buf)
	err = enc.Encode(result)
	if err != nil {
		panic(errors.NewErr("[nativeInvoke]AppCall failed:" + err.Error()))
	}

	buf = bytes.NewBuffer(nil)
	bs := buf.Bytes()
	self.CallOutPut = make([]byte, len(bs))

	copy(self.CallOutPut, bs)

	return uint32(len(self.CallOutPut))
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

func NewHostModule(host *Runtime) *wasm.Module {
	m := wasm.NewModule()
	m.Types = &wasm.SectionTypes{
		Entries: []wasm.FunctionSig{
			{
				Form:       0, // value for the 'func' type constructor
				ParamTypes: []wasm.ValueType{wasm.ValueTypeI32},
			},
			{
				Form:        0, // value for the 'func' type constructor
				ReturnTypes: []wasm.ValueType{wasm.ValueTypeI32},
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
			"self_address": {
				FieldStr: "self_address",
				Kind:     wasm.ExternalFunction,
				Index:    3,
			},
			"caller_address": {
				FieldStr: "caller_address",
				Kind:     wasm.ExternalFunction,
				Index:    4,
			},
			"entry_address": {
				FieldStr: "entry_address",
				Kind:     wasm.ExternalFunction,
				Index:    5,
			},
			"get_input": {
				FieldStr: "get_input",
				Kind:     wasm.ExternalFunction,
				Index:    6,
			},
			"get_callout": {
				FieldStr: "get_callout",
				Kind:     wasm.ExternalFunction,
				Index:    7,
			},
			"check_witness": {
				FieldStr: "check_witness",
				Kind:     wasm.ExternalFunction,
				Index:    8,
			},
			"get_current_blockhash": {
				FieldStr: "get_current_blockhash",
				Kind:     wasm.ExternalFunction,
				Index:    9,
			},
			"get_current_txhash": {
				FieldStr: "get_current_txhash",
				Kind:     wasm.ExternalFunction,
				Index:    10,
			},
			"ret": {
				FieldStr: "ret",
				Kind:     wasm.ExternalFunction,
				Index:    11,
			},
			"notify": {
				FieldStr: "notify",
				Kind:     wasm.ExternalFunction,
				Index:    12,
			},
			"call_contract": {
				FieldStr: "call_contract",
				Kind:     wasm.ExternalFunction,
				Index:    13,
			},
			"storage_read": {
				FieldStr: "storage_read",
				Kind:     wasm.ExternalFunction,
				Index:    14,
			},
			"storage_write": {
				FieldStr: "storage_write",
				Kind:     wasm.ExternalFunction,
				Index:    15,
			},
			"contract_create": {
				FieldStr: "contract_create",
				Kind:     wasm.ExternalFunction,
				Index:    16,
			},
			"contract_migrate": {
				FieldStr: "contract_migrate",
				Kind:     wasm.ExternalFunction,
				Index:    17,
			},
			"contract_delete": {
				FieldStr: "contract_delete",
				Kind:     wasm.ExternalFunction,
				Index:    18,
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

	if dep.NeedStorage == byte(3) {
		return WASMVM_CONTRACT, nil
	}

	return NEOVM_CONTRACT, nil

}

func serializeStorageKey(contractAddress common.Address, key []byte) ([]byte, error) {
	bf := new(bytes.Buffer)
	storageKey := &states2.StorageKey{ContractAddress: contractAddress, Key: key}
	if _, err := storageKey.Serialize(bf); err != nil {
		return []byte{}, errors.NewErr("[serializeStorageKey] StorageKey serialize error!")
	}
	return bf.Bytes(), nil
}
