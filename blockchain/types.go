package blockchain

import "github.com/pkg/errors"

var (
	ErrorNotSuccess = errors.New("Error: Not success")
)

type BlockInfo struct {
	Height    uint64
	Timestamp uint64
	Hash      string
	TxHashes  []string
}

type BlockTxInfo struct {
	Height   uint64
	TxHashes []string
}
