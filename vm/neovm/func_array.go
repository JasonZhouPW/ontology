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
	"math/big"

	"github.com/ontio/ontology/vm/neovm/types"
	"fmt"
)

func opArraySize(e *ExecutionEngine) (VMState, error) {
	item := PopStackItem(e)
	if _, ok := item.(*types.Array); ok {
		PushData(e, len(item.GetArray()))
	} else {
		PushData(e, len(item.GetByteArray()))
	}

	return NONE, nil
}

func opPack(e *ExecutionEngine) (VMState, error) {
	size := PopInt(e)
	var items []types.StackItems
	for i := 0; i < size; i++ {
		items = append(items, PopStackItem(e))
	}
	PushData(e, items)
	return NONE, nil
}

func opUnpack(e *ExecutionEngine) (VMState, error) {
	arr := PopArray(e)
	l := len(arr)
	for i := l - 1; i >= 0; i-- {
		Push(e, arr[i])
	}
	PushData(e, l)
	return NONE, nil
}

/*func opPickItem(e *ExecutionEngine) (VMState, error) {
	index := PopInt(e)
	items := PopArray(e)
	PushData(e, items[index])
	return NONE, nil
}
*/
func opPickItem(e *ExecutionEngine) (VMState, error) {
	index := PopStackItem(e)
	items := PopStackItem(e)

	switch t:= items.(type){
	case *types.Array:
		fmt.Println("opPickItem array!!!!")
		fmt.Printf("index is %v\n",index)
		PushData(e, items.GetArray()[int(index.GetBigInteger().Int64())])
		fmt.Printf("pickitem  %v\n",items.GetArray()[int(index.GetBigInteger().Int64())])
	case *types.Map:
		fmt.Println("opPickItem map!!!!")
		fmt.Printf("index is %v\n",index)
		fmt.Printf("value is %v\n",items.GetMap()[index])

		PushData(e,items.GetMap()[index])
	default:
		fmt.Printf("opPickItem:t is %v\n",t)
	}


	return NONE, nil
}



func opSetItem(e *ExecutionEngine) (VMState, error) {

	printEvaStack(e)
	printAltStack(e)


	newItem := PopStackItem(e)
	if value, ok := newItem.(*types.Struct); ok {
		newItem = value.Clone()
	}

	index := PopStackItem(e)
	item := PopStackItem(e)

	switch t:=item.(type){
	case *types.Map:

		fmt.Printf("====opSetItem :value is %v\n",newItem)
	    mapitem := item.GetMap()
		fmt.Printf("====opSetItem: key is %v,value is %v\n",index,newItem)
		fmt.Printf("====opSetItem: key is %s,value is %s\n",index.GetByteArray(),newItem.GetByteArray())

		fmt.Printf("====opSetItem: mapitem is %v\n",mapitem)
		mapitem[index] = newItem

	case *types.Array:
		fmt.Printf("====opSetItem :value is %v\n",newItem)
		items := item.GetArray()
		fmt.Printf(" ====opSetItem index:%d,items:%v\n",index,items)
		items[int(index.GetBigInteger().Int64())] = newItem
	default:
		fmt.Printf("====opSetItem default is %v\n",t)

	}

	return NONE, nil
}
/*
func opSetItem(e *ExecutionEngine) (VMState, error) {
	newItem := PopStackItem(e)
	if value, ok := newItem.(*types.Struct); ok {
		newItem = value.Clone()
	}

	switch newItem.(type) {
	case *types.Array:
		index := PopInt(e)
		items := PopArray(e)
		items[index] = newItem
	case *types.Map:
		key := PopStackItem(e)
		mapitem := PopMap(e)
		mapitem[key] = newItem
		fmt.Printf("====opSetItem: key is %v,value is %v\n",key,newItem)
		fmt.Printf("====opSetItem: mapitem is %v\n",mapitem)
	}
	return NONE, nil
}*/

func opNewArray(e *ExecutionEngine) (VMState, error) {
	count := PopInt(e)
	var items []types.StackItems
	for i := 0; i < count; i++ {
		items = append(items, types.NewBoolean(false))
	}
	PushData(e, types.NewArray(items))
	return NONE, nil
}

func opNewStruct(e *ExecutionEngine) (VMState, error) {
	count := PopBigInt(e)
	var items []types.StackItems
	for i := 0; count.Cmp(big.NewInt(int64(i))) > 0; i++ {
		items = append(items, types.NewBoolean(false))
	}
	PushData(e, types.NewStruct(items))
	return NONE, nil
}

func opNewMap(e *ExecutionEngine) (VMState, error) {
	//count := PopBigInt(e)
	//fmt.Printf("opNewMap...count.%d\n",count)
	PushData(e, types.NewMap())
	return NONE, nil
}


func opAppend(e *ExecutionEngine) (VMState, error) {
	newItem := PopStackItem(e)
	if value, ok := newItem.(*types.Struct); ok {
		newItem = value.Clone()
	}
	itemArr := PopArray(e)
	itemArr = append(itemArr, newItem)
	return NONE, nil
}

func opReverse(e *ExecutionEngine) (VMState, error) {
	itemArr := PopArray(e)
	for i, j := 0, len(itemArr)-1; i < j; i, j = i+1, j-1 {
		itemArr[i], itemArr[j] = itemArr[j], itemArr[i]
	}
	return NONE, nil
}

//only for test
func printStackItem(item types.StackItems){
	switch item.(type){
	case *types.Boolean:
		fmt.Printf("boolean :%v\n",item.GetBoolean())
	case *types.Map:
		fmt.Printf("map :%v\n",item.GetMap())
	case *types.Array:
		fmt.Printf("Array :%v\n",item.GetArray())
	case *types.ByteArray:
		fmt.Printf("ByteArray :%v\n",item.GetByteArray())
	case *types.Integer:
		fmt.Printf("Integer :%v\n",item.GetBigInteger().Int64())

	default:

		fmt.Println("skip..")
	}
}

func printEvaStack(e *ExecutionEngine){
	fmt.Println("======printEvaStack start=")
	for _,item := range e.EvaluationStack.e{
		printStackItem(item)
	}
	fmt.Println("======printEvaStack end=")

}

func printAltStack(e *ExecutionEngine){
	fmt.Println("======printAltStack start=")
	for _,item := range e.AltStack.e{
		printStackItem(item)
	}
	fmt.Println("======printAltStack end=")

}