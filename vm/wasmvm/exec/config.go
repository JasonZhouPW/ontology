package exec

import "sync"

var (
	//Gas Limit
	MIN_TRANSACTION_GAS           uint64 = 20000 // Per transaction base cost.
	BLOCKCHAIN_GETHEADER_GAS      uint64 = 100
	BLOCKCHAIN_GETBLOCK_GAS       uint64 = 200
	BLOCKCHAIN_GETTRANSACTION_GAS uint64 = 100
	BLOCKCHAIN_GETCONTRACT_GAS    uint64 = 100
	CONTRACT_CREATE_GAS           uint64 = 20000000
	CONTRACT_MIGRATE_GAS          uint64 = 20000000
	UINT_DEPLOY_CODE_LEN_GAS      uint64 = 200000
	UINT_INVOKE_CODE_LEN_GAS      uint64 = 20000
	NATIVE_INVOKE_GAS             uint64 = 1000
	STORAGE_GET_GAS               uint64 = 200
	STORAGE_PUT_GAS               uint64 = 4000
	STORAGE_DELETE_GAS            uint64 = 100
	RUNTIME_CHECKWITNESS_GAS      uint64 = 200
	RUNTIME_ADDRESSTOBASE58_GAS   uint64 = 40
	RUNTIME_BASE58TOADDRESS_GAS   uint64 = 30
	APPCALL_GAS                   uint64 = 10
	TAILCALL_GAS                  uint64 = 10
	SHA1_GAS                      uint64 = 10
	SHA256_GAS                    uint64 = 10
	HASH160_GAS                   uint64 = 20
	HASH256_GAS                   uint64 = 20
	OPCODE_GAS                    uint64 = 1

	//BLOCKCHAIN_GETHEADER_NAME      = ""
	//BLOCKCHAIN_GETBLOCK_NAME	   = ""
	//BLOCKCHAIN_GETTRANSACTION_NAME = ""
	//BLOCKCHAIN_GETCONTRACT_NAME    = ""
	BLOCK_GETHEADERHASH_NAME        = "ONT_Block_GetCurrentHeaderHash"
	BLOCK_GETHEADERHEIGHT_NAME      = "ONT_Block_GetCurrentHeaderHeight"
	BLOCK_GETBLOCKHASH_NAME         = "ONT_Block_GetCurrentBlockHash"
	BLOCK_GETBLOCKHEIGHT_NAME       = "ONT_Block_GetCurrentBlockHeight"
	BLOCK_GETTRANSACTIONBYHASH_NAME = "ONT_Block_GetTransactionByHash"
	BLOCK_GETTRANSACTIONCOUNT_NAME  = "ONT_Block_GetTransactionCount"
	BLOCK_GETTRANSACTIONS           = "ONT_Block_GetTransactions"

	BLOCKCHAIN_GETHEGITH         = "ONT_BlockChain_GetHeight"
	BLOCKCHAIN_GETHEADERBYHEIGHT = "ONT_BlockChain_GetHeaderByHeight"
	BLOCKCHAIN_GETHEADERBYHASH   = "ONT_BlockChain_GetHeaderByHash"
	BLOCKCHAIN_GETBLOCKBYHEIGHT  = "ONT_BlockChain_GetBlockByHeight"
	BLOCKCHAIN_GETBLOCKBYHASH    = "ONT_BlockChain_GetBlockByHash"
	BLOCKCHAIN_GETCONTRACT       = "ONT_BlockChain_GetContract"

	HEADER_GETHASH          = "ONT_Header_GetHash"
	HEADER_GETVERSION       = "ONT_Header_GetVersion"
	HEADER_GETPREVHASH      = "ONT_Header_GetPrevHash"
	HEADER_GETMERKLEROOT    = "ONT_Header_GetMerkleRoot"
	HEADER_GETINDEX         = "ONT_Header_GetIndex"
	HEADER_GETTIMESTAMP     = "ONT_Header_GetTimestamp"
	HEADER_GETCONSENSUSDATA = "ONT_Header_GetConsensusData"
	HEADER_GETNEXTCONSENSUS = "ONT_Header_GetNextConsensus"

	TRANSACTION_GETHASH       = "ONT_Transaction_GetHash"
	TRANSACTION_GETTYPE       = "ONT_Transaction_GetType"
	TRANSACTION_GETATTRIBUTES = "ONT_Transaction_GetAttributes"

	CONTRACT_CREATE_NAME       = "Ontology.Contract.Create"
	CONTRACT_MIGRATE_NAME      = "Ontology.Contract.Migrate"
	WASM_CONTRACT_CREATE_NAME  = "ONT_Contract_Create"
	WASM_CONTRACT_MIGRATE_NAME = "ONT_Contract_Migrate"
	WASM_CONTRACT_DELETE_NAME  = "ONT_Contract_Delete"
	STORAGE_GET_NAME           = "ONT_Storage_Get"
	STORAGE_PUT_NAME           = "ONT_Storage_Put"
	STORAGE_DELETE_NAME        = "ONT_Storage_Delete"
	RUNTIME_CHECKWITNESS_NAME  = "ONT_Runtime_CheckWitness"
	NATIVE_INVOKE_NAME         = "ONT_NativeInvoke"
	APPCALL_NAME               = "ONT_CallContract"
	//TAILCALL_NAME                  = ""
	SHA1_NAME                 = "SHA1"
	SHA256_NAME               = "SHA256"
	HASH160_NAME              = "Hash160"
	HASH256_NAME              = "Hash256"
	UINT_DEPLOY_CODE_LEN_NAME = "Deploy.Code.Gas"
	UINT_INVOKE_CODE_LEN_NAME = "Invoke.Code.Gas"
	//RUNTIME_BASE58TOADDRESS_NAME   = ""
	//RUNTIME_ADDRESSTOBASE58_NAME   = ""

	PER_UNIT_CODE_LEN    int = 1024
	METHOD_LENGTH_LIMIT  int = 1024
	DUPLICATE_STACK_SIZE int = 1024 * 2
	VM_STEP_LIMIT        int = 400000

	GAS_TABLE = initGAS_TABLE()
)

func initGAS_TABLE() *sync.Map {
	m := sync.Map{}
	m.Store(BLOCKCHAIN_GETHEGITH, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(BLOCKCHAIN_GETHEADERBYHEIGHT, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(BLOCKCHAIN_GETHEADERBYHASH, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(BLOCKCHAIN_GETBLOCKBYHEIGHT, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(BLOCKCHAIN_GETBLOCKBYHASH, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(BLOCKCHAIN_GETCONTRACT, BLOCKCHAIN_GETHEADER_GAS)

	m.Store(BLOCK_GETHEADERHASH_NAME, BLOCKCHAIN_GETBLOCK_GAS)
	m.Store(BLOCK_GETHEADERHEIGHT_NAME, BLOCKCHAIN_GETBLOCK_GAS)
	m.Store(BLOCK_GETBLOCKHASH_NAME, BLOCKCHAIN_GETBLOCK_GAS)
	m.Store(BLOCK_GETBLOCKHEIGHT_NAME, BLOCKCHAIN_GETBLOCK_GAS)
	m.Store(BLOCK_GETTRANSACTIONBYHASH_NAME, BLOCKCHAIN_GETBLOCK_GAS)
	m.Store(BLOCK_GETTRANSACTIONCOUNT_NAME, BLOCKCHAIN_GETBLOCK_GAS)
	m.Store(BLOCK_GETTRANSACTIONS, BLOCKCHAIN_GETBLOCK_GAS)

	m.Store(HEADER_GETHASH, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(HEADER_GETVERSION, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(HEADER_GETPREVHASH, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(HEADER_GETMERKLEROOT, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(HEADER_GETINDEX, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(HEADER_GETTIMESTAMP, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(HEADER_GETCONSENSUSDATA, BLOCKCHAIN_GETHEADER_GAS)
	m.Store(HEADER_GETNEXTCONSENSUS, BLOCKCHAIN_GETHEADER_GAS)

	m.Store(TRANSACTION_GETHASH, BLOCKCHAIN_GETTRANSACTION_GAS)
	m.Store(TRANSACTION_GETTYPE, BLOCKCHAIN_GETTRANSACTION_GAS)
	m.Store(TRANSACTION_GETATTRIBUTES, BLOCKCHAIN_GETTRANSACTION_GAS)

	m.Store(CONTRACT_CREATE_NAME, CONTRACT_CREATE_GAS)
	m.Store(CONTRACT_MIGRATE_NAME, CONTRACT_MIGRATE_GAS)
	m.Store(WASM_CONTRACT_CREATE_NAME, CONTRACT_CREATE_GAS)
	m.Store(WASM_CONTRACT_MIGRATE_NAME, CONTRACT_MIGRATE_GAS)
	m.Store(STORAGE_GET_NAME, STORAGE_GET_GAS)
	m.Store(STORAGE_PUT_NAME, STORAGE_PUT_GAS)
	m.Store(STORAGE_DELETE_NAME, STORAGE_DELETE_GAS)
	m.Store(RUNTIME_CHECKWITNESS_NAME, RUNTIME_CHECKWITNESS_GAS)
	m.Store(NATIVE_INVOKE_NAME, NATIVE_INVOKE_GAS)
	m.Store(APPCALL_NAME, APPCALL_GAS)
	//m.Store(TAILCALL_NAME, TAILCALL_GAS)
	m.Store(SHA1_NAME, SHA1_GAS)
	m.Store(SHA256_NAME, SHA256_GAS)
	m.Store(HASH160_NAME, HASH160_GAS)
	m.Store(HASH256_NAME, HASH256_GAS)
	m.Store(UINT_DEPLOY_CODE_LEN_NAME, UINT_DEPLOY_CODE_LEN_GAS)
	m.Store(UINT_INVOKE_CODE_LEN_NAME, UINT_INVOKE_CODE_LEN_GAS)

	//m.Store(RUNTIME_BASE58TOADDRESS_NAME, RUNTIME_BASE58TOADDRESS_GAS)
	//m.Store(RUNTIME_ADDRESSTOBASE58_NAME, RUNTIME_ADDRESSTOBASE58_GAS)

	return &m
}
