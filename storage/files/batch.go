package files

import (
	"context"

	"github.com/0glabs/0g-storage-client/node"
	"github.com/openweb3/go-rpc-provider"
	"github.com/openweb3/go-rpc-provider/interfaces"
	"github.com/pkg/errors"
)

type batchGetFileInfoResult struct {
	files  []*node.FileInfo
	errors []error
}

func batchGetFileInfo(provider interfaces.Provider, ctx context.Context, txSeqs ...uint64) (*batchGetFileInfoResult, error) {
	var batch []rpc.BatchElem

	for _, v := range txSeqs {
		batch = append(batch, rpc.BatchElem{
			Method: "zgs_getFileInfoByTxSeq",
			Args:   []interface{}{v},
			Result: &node.FileInfo{},
		})
	}

	if err := provider.BatchCallContext(ctx, batch); err != nil {
		return nil, errors.WithMessage(err, "Failed to batch call RPC")
	}

	result := batchGetFileInfoResult{
		files:  make([]*node.FileInfo, 0, len(batch)),
		errors: make([]error, 0, len(batch)),
	}

	for _, v := range batch {
		if v.Error != nil {
			result.files = append(result.files, nil)
			result.errors = append(result.errors, v.Error)
		} else if ret := v.Result.(*node.FileInfo); ret.Tx.Size > 0 {
			result.files = append(result.files, ret)
			result.errors = append(result.errors, nil)
		} else {
			result.files = append(result.files, nil)
			result.errors = append(result.errors, nil)
		}
	}

	return &result, nil
}
