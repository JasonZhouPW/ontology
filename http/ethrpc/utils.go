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
package ethrpc

import (
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	types2 "github.com/ethereum/go-ethereum/core/types"
	oComm "github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
)

func EthBlockFromOntology(block *types.Block, fullTx bool) map[string]interface{} {
	if block == nil {
		return nil
	}
	hash := block.Hash()
	transactions, gasUsed, ethTxs := EthTransactionsFromOntology(block.Transactions, common.BytesToHash(hash.ToArray()), uint64(block.Header.Height))

	var blockTxs interface{}

	if fullTx {
		blockTxs = ethTxs
	} else {
		blockTxs = transactions
	}
	return FormatBlock(*block, 0, gasUsed, blockTxs)
}

func EthTransactionsFromOntology(txs []*types.Transaction, blockHash common.Hash, blockNumber uint64) ([]common.Hash, *big.Int, []*types2.Transaction) {
	var transactionHashes []common.Hash
	var transactions []*types2.Transaction
	gasUsed := big.NewInt(0)
	for idx, tx := range txs {
		hash := tx.Hash()
		ethTx, err := OntTxToEthTx(*tx, blockHash, blockNumber, uint64(idx))
		if err != nil {
			continue
		}
		gasUsed.Add(gasUsed, big.NewInt(int64(ethTx.Gas())))
		transactionHashes = append(transactionHashes, common.BytesToHash(hash.ToArray()))
		transactions = append(transactions, ethTx)
	}
	return transactionHashes, gasUsed, transactions
}

func OntTxToEthTx(tx types.Transaction, blockHash common.Hash, blockNumber, index uint64) (*types2.Transaction, error) {
	eip155Tx, err := tx.GetEIP155Tx()
	if err != nil {
		return nil, err
	}
	return eip155Tx, nil
}

func FormatBlock(block types.Block, gasLimit uint64, gasUsed *big.Int, transactions interface{}) map[string]interface{} {
	size := len(block.ToArray())
	header := block.Header
	hash := header.Hash()
	ret := map[string]interface{}{
		"number":           hexutil.Uint64(header.Height),
		"hash":             hexutil.Bytes(hash[:]),
		"parentHash":       hexutil.Bytes(header.PrevBlockHash[:]),
		"nonce":            types2.BlockNonce{}, // PoW specific
		"sha3Uncles":       common.Hash{},       // No uncles in Tendermint
		"logsBloom":        types2.Bloom{},
		"transactionsRoot": hexutil.Bytes(header.TransactionsRoot[:]),
		"stateRoot":        hexutil.Bytes{},
		"miner":            common.Address{},
		"mixHash":          common.Hash{},
		"difficulty":       hexutil.Uint64(0),
		"totalDifficulty":  hexutil.Uint64(0),
		"extraData":        hexutil.Bytes{},
		"size":             hexutil.Uint64(size),
		"gasLimit":         hexutil.Uint64(gasLimit), // TODO Static gas limit
		"gasUsed":          (*hexutil.Big)(gasUsed),
		"timestamp":        hexutil.Uint64(header.Timestamp),
		"uncles":           []string{},
		"receiptsRoot":     common.Hash{},
	}
	if !reflect.ValueOf(transactions).IsNil() {
		switch transactions.(type) {
		case []common.Hash:
			ret["transactions"] = transactions.([]common.Hash)
		case []*types2.Transaction:
			ret["transactions"] = transactions.([]*types2.Transaction)
		}
	} else {
		ret["transactions"] = []common.Hash{}
	}
	return ret
}

func EthToOntAddr(address common.Address) oComm.Address {
	return oComm.Address(address)
}
