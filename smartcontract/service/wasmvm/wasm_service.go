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
	"fmt"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/wasm"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/serialization"
	"github.com/ontio/ontology/core/store"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/context"
	"github.com/ontio/ontology/smartcontract/event"
	"github.com/ontio/ontology/smartcontract/service/neovm"
	"github.com/ontio/ontology/smartcontract/states"
	"github.com/ontio/ontology/smartcontract/storage"
)

type WasmVmService struct {
	Store         store.LedgerStore
	CacheDB       *storage.CacheDB
	ContextRef    context.ContextRef
	Notifications []*event.NotifyEventInfo
	Code          []byte
	Tx            *types.Transaction
	Time          uint32
	Height        uint32
	BlockHash     common.Uint256
	PreExec       bool
	GasPrice      uint64
	GasLimit      uint64
	IsTerminate   bool
}

var (
	ERR_CHECK_STACK_SIZE  = errors.NewErr("[WasmVmService] vm over max stack size!")
	ERR_EXECUTE_CODE      = errors.NewErr("[WasmVmService] vm execute code invalid!")
	ERR_GAS_INSUFFICIENT  = errors.NewErr("[WasmVmService] gas insufficient")
	VM_EXEC_STEP_EXCEED   = errors.NewErr("[WasmVmService] vm execute step exceed!")
	CONTRACT_NOT_EXIST    = errors.NewErr("[WasmVmService] Get contract code from db fail")
	DEPLOYCODE_TYPE_ERROR = errors.NewErr("[WasmVmService] DeployCode type error!")
	VM_EXEC_FAULT         = errors.NewErr("[WasmVmService] vm execute state fault!")
	VM_INIT_FAULT         = errors.NewErr("[WasmVmService] vm init state fault!")

	CONTRACT_METHOD_NAME = "invoke"
)

func (this *WasmVmService) Invoke() (interface{}, error) {
	if len(this.Code) == 0 {
		return nil, ERR_EXECUTE_CODE
	}

	contract := &states.ContractInvokeParam{}
	contract.Deserialize(bytes.NewBuffer(this.Code))

	code, err := this.Store.GetContractState(contract.Address)
	if err != nil {
		return nil, err
	}

	this.ContextRef.PushContext(&context.Context{ContractAddress: contract.Address, Code: code.Code})

	bf := bytes.NewBuffer(nil)
	serialization.WriteString(bf, contract.Method)
	bf.Write(contract.Args)

	host := &Runtime{Service: this, Input: bf.Bytes()}

	m, err := wasm.ReadModule(bytes.NewReader(code.Code), func(name string) (*wasm.Module, error) {
		switch name {
		case "env":
			return NewHostModule(host), nil
		}
		return nil, fmt.Errorf("module %q unknown", name)
	})
	if err != nil {
		return nil, err
	}

	if m.Export == nil {
		return nil, errors.NewErr("[Call]No export in wasm!")
	}

	vm, err := exec.NewVM(m)
	if err != nil {
		return nil, VM_INIT_FAULT
	}
	if this.PreExec {
		this.GasLimit = uint64(neovm.VM_STEP_LIMIT)
	}
	vm.RecoverPanic = true
	vm.AvaliableGas = &exec.Gas{GasLimit: this.GasLimit, GasPrice: this.GasPrice}

	entryName := CONTRACT_METHOD_NAME

	entry, ok := m.Export.Entries[entryName]

	if ok == false {
		return nil, errors.NewErr("[Call]Method:" + entryName + " does not exist!")
	}

	//get entry index
	index := int64(entry.Index)

	//get function index
	fidx := m.Function.Types[int(index)]

	//get  function type
	ftype := m.Types.Entries[int(fidx)]

	//no returns of the entry function
	if len(ftype.ReturnTypes) > 0 {
		return nil, errors.NewErr("[Call]ExecCode error! Invoke function sig error")
	}

	//nor args for passed in, all args in runtime input buffer

	_, err = vm.ExecCode(index)
	if err != nil {
		return nil, errors.NewErr("[Call]ExecCode error!" + err.Error())
	}

	return host.Output, nil
}
