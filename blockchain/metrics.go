package blockchain

const (
	// node level
	// block
	blockCollatedGapPattern         = "monitor/blockchain/block/gap/%v"
	blockCollatedGapUnhealthPattern = "monitor/blockchain/block/gap/unhealth/%v"

	blockHeightBehindPattern   = "monitor/blockchain/block/height/behind/%v"
	blockHeightUnhealthPattern = "monitor/blockchain/block/height/unhealth/%v"

	// count of tx in block
	blockTxCountPattern = "monitor/blockchain/block/tx/count"

	// chain
	chainForkPattern = "monitor/blockchain/fork/%v"
	// ethermint rpc
	nodeEthRpcLatencyPattern  = "monitor/blockchain/rpc/ethermint/latency/%v"
	nodeEthRpcUnhealthPattern = "monitor/blockchain/rpc/ethermint/unhealth/%v"
	// cosmos rpc
	nodeCosmosRpcLatencyPattern  = "monitor/blockchain/rpc/cosmos/latency/%v"
	nodeCosmosRpcUnhealthPattern = "monitor/blockchain/rpc/cosmos/unhealth/%v"
	// cometbft rpc
	nodeCometbftRpcLatencyPattern  = "monitor/blockchain/rpc/cometbft/latency/%v"
	nodeCometbftRpcUnhealthPattern = "monitor/blockchain/rpc/cometbft/unhealth/%v"

	// mempool
	mempoolUncommitTxCntPattern = "monitor/blockchain/mempool/uncommit/cnt"
	mempoolHighLoadPattern      = "monitor/blockchain/mempool/highload"

	// validator
	validatorActiveCountUnhealthPattern = "monitor/blockchain/validator/count/unhealth"
	validatorActiveCountPattern         = "monitor/blockchain/validator/count"
	validatorJailedPattern              = "monitor/blockchain/validator/jailed/%v"

	// tx
	failedTxCountPattern         = "monitor/blockchain/tx/failed/count"
	failedTxCountUnhealthPattern = "monitor/blockchain/tx/failed/count/unhealth"

	// chain height
	chainHeightHaltPattern = "monitor/blockchain/height/halt"
)
