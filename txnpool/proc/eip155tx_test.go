package proc

import (
	"crypto/ecdsa"
	"fmt"
	ethcomm "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ontio/ontology/common"
	txtypes "github.com/ontio/ontology/core/types"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func Test_GenEIP155tx(t *testing.T) {
	privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	assert.Nil(t, err)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		assert.True(t, ok)
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Printf("addr:%s\n", fromAddress.Hex())

	ontAddress, err := common.AddressParseFromBytes(fromAddress[:])
	assert.Nil(t, err)
	fmt.Printf("ont addr:%s\n", ontAddress.ToBase58())

	value := big.NewInt(1000000000)
	gaslimit := uint64(21000)
	gasPrice := big.NewInt(2500)
	nonce := uint64(0)

	toAddress := ethcomm.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")

	var data []byte
	tx := types.NewTransaction(nonce, toAddress, value, gaslimit, gasPrice, data)

	chainId := big.NewInt(0)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainId), privateKey)
	assert.Nil(t, err)

	otx, err := txtypes.TransactionFromEIP155(signedTx)
	assert.Nil(t, err)

	assert.True(t, otx.TxType == txtypes.EIP155)

}
