package types

import (
	"reflect"
	"math/big"
	"github.com/ontio/ontology/vm/neovm/interfaces"
	"bytes"
)

type Map struct{
	_map map[StackItems]StackItems
}

func NewMap() *Map{
	return &Map{make(map[StackItems]StackItems)}
}

func (this *Map)NewValue(v map[StackItems]StackItems){
	this._map = v
}

func (this *Map)Add(key StackItems,value StackItems){
	this._map[key] = value
}

func (this *Map)Clear(){
	this._map = make(map[StackItems]StackItems)
}

func (this *Map)ContainsKey(key StackItems)bool{
	_,ok := this._map[key]
	return ok
}

func (this *Map)Remove(key StackItems){
	delete(this._map, key)
}

func (this *Map)Equals(that StackItems) bool{
	return reflect.DeepEqual(this,that)
}

func (this *Map)GetBoolean() bool{
	return true
}

func (this *Map)GetByteArray() []byte{
	return this.ToArray()
}

func (this *Map)GetBigInteger() *big.Int  {
	return nil
}

func (this *Map)GetInterface() interfaces.Interop{
	return nil
}

func (this *Map)GetArray() []StackItems{
	return nil
}

func(this *Map)GetStruct() []StackItems{
	return nil
}

func (this *Map)GetMap() map[StackItems]StackItems{
	return this._map
}

func (this *Map)ToArray() []byte{
	bf := bytes.NewBuffer(nil)
	i := 0
	l := len(this._map)
	for k,v :=range this._map{
		bf.WriteString("[")
		bf.Write(k.GetByteArray())
		bf.WriteString(":")
		bf.Write(v.GetByteArray())
		bf.WriteString("]")
		if i < l{
			bf.WriteString(",")
			i++
		}
	}
	return bf.Bytes()
}