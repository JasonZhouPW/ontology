package types

import (
	"testing"
	"math/big"
)

func TestConvertBigIntegerToBytes(t *testing.T) {
	i := big.NewInt(200000)

	b := ConvertBigIntegerToBytes(i)

	j := ConvertBytesToBigInteger(b)

	if i.Int64() != j.Int64(){
		t.Error("i should equals j")
	}
}