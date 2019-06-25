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
package neovm

import (
	"fmt"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/smartcontract/service/util"
	"github.com/ontio/ontology/smartcontract/states"
	vm "github.com/ontio/ontology/vm/neovm"
)

//neovm contract call wasmvm contract
func WASMInvoke(service *NeoVmService, engine *vm.ExecutionEngine) error {
	count := vm.EvaluationStackCount(engine)
	if count < 1 {
		return fmt.Errorf("invoke wasm contract invalid parameters %d < 1 ", count)
	}
	parambytes, err := vm.PopByteArray(engine)
	if err != nil {
		return err
	}
	list, err := util.DeserializeInput(parambytes)
	if err != nil {
		return err
	}
	if len(list) < 1 {
		return fmt.Errorf("need wasm contract address")
	}

	contractAddress, ok := list[0].(common.Address)
	if !ok {
		return fmt.Errorf("first param should be wasm contract address")
	}

	dp, err := service.CacheDB.GetContract(contractAddress)
	if err != nil {
		return err
	}
	if dp == nil {
		return fmt.Errorf("wasm contract does not exist")
	}
	if dp.VmType != payload.WASMVM_TYPE {
		return fmt.Errorf("not a wasm contract")
	}

	inputs, err := util.BuildWasmVMInvokeCode(contractAddress, list[1:])
	if err != nil {
		return err
	}

	conParam := states.WasmContractParam{Address: contractAddress, Args: inputs}
	sink := common.NewZeroCopySink(nil)
	conParam.Serialization(sink)

	newservice, err := service.ContextRef.NewExecuteEngine(sink.Bytes(), types.InvokeWasm)
	if err != nil {
		return err
	}

	tmpRes, err := newservice.Invoke()
	if err != nil {
		return err
	}
	vm.PushData(engine, tmpRes.([]byte))
	return nil

}
