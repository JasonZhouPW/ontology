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
	"bytes"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
	"github.com/ontio/ontology/smartcontract/states"
	vm "github.com/ontio/ontology/vm/neovm"
)

// GetCodeContainer push current transaction to vm stack
func GetCodeContainer(service *NeoVmService, engine *vm.ExecutionEngine) error {
	vm.PushData(engine, service.Tx)
	return nil
}

// GetExecutingAddress push current context to vm stack
func GetExecutingAddress(service *NeoVmService, engine *vm.ExecutionEngine) error {
	context := service.ContextRef.CurrentContext()
	if context == nil {
		return errors.NewErr("Current context invalid")
	}
	vm.PushData(engine, context.ContractAddress[:])
	return nil
}

// GetExecutingAddress push previous context to vm stack
func GetCallingAddress(service *NeoVmService, engine *vm.ExecutionEngine) error {
	context := service.ContextRef.CallingContext()
	if context == nil {
		return errors.NewErr("Calling context invalid")
	}
	vm.PushData(engine, context.ContractAddress[:])
	return nil
}

// GetExecutingAddress push entry call context to vm stack
func GetEntryAddress(service *NeoVmService, engine *vm.ExecutionEngine) error {
	context := service.ContextRef.EntryContext()
	if context == nil {
		return errors.NewErr("Entry context invalid")
	}
	vm.PushData(engine, context.ContractAddress[:])
	return nil
}

//serialize contract to bytes
func SerializeContract(service *NeoVmService, engine *vm.ExecutionEngine) error {

/*	version := vm.PopBigInt(engine).Int64()
	code := vm.PopByteArray(engine)
	contractAddr := vm.PopByteArray(engine)
	method := vm.PopByteArray(engine)
	args := vm.PopByteArray(engine)*/
	args := vm.PopByteArray(engine)
	method := vm.PopByteArray(engine)
	contractAddr := vm.PopByteArray(engine)
	code := vm.PopByteArray(engine)
	version := vm.PopBigInt(engine).Int64()

	addrbytes, err := common.HexToBytes(string(contractAddr))
	if err != nil {
		return err
	}
	address, err := common.AddressParseFromBytes(addrbytes)
	if err != nil {
		return err
	}

	if len(code) == 0{
		code = nil
	}

	contract := &states.Contract{Version: byte(version),
		Code:    code,
		Address: address,
		Method:  string(method),
		Args:    args}

	bf := bytes.NewBuffer(nil)
	err = contract.Serialize(bf)
	if err != nil {
		return err
	}

	vm.PushData(engine, bf.Bytes())

	return nil
}

func SerializeTransfer(service *NeoVmService, engine *vm.ExecutionEngine) error {

	from := vm.PopByteArray(engine)
	to := vm.PopByteArray(engine)
	amount := vm.PopBigInt(engine).Int64()

	fAddr, err := common.AddressParseFromBytes(from)
	if err != nil {
		return err
	}
	tAddr, err := common.AddressParseFromBytes(to)
	if err != nil {
		return err
	}

	state := &ont.State{From: fAddr,
		To:    tAddr,
		Value: uint64(amount)}

	tranfer := ont.Transfers{
		States: []*ont.State{state},
	}
	bf := bytes.NewBuffer(nil)
	err = tranfer.Serialize(bf)
	if err != nil {
		return err
	}

	vm.PushData(engine, bf.Bytes())

	return nil
}
