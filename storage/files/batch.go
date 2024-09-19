package files

import (
	"context"

	"github.com/openweb3/go-rpc-provider"
	"github.com/openweb3/go-rpc-provider/interfaces"
)

type BatchRpcRequest struct {
	Method string
	Args   []any
}

type BatchRpcResponse[T any] struct {
	Result T
	Error  error
}

func BatchCallRpc[T any](provider interfaces.Provider, ctx context.Context, requests ...BatchRpcRequest) ([]BatchRpcResponse[T], error) {
	batch := make([]rpc.BatchElem, 0, len(requests))
	responses := make([]BatchRpcResponse[T], len(requests))

	for i, v := range requests {
		batch = append(batch, rpc.BatchElem{
			Method: v.Method,
			Args:   v.Args,
			Result: &responses[i].Result,
		})
	}

	if err := provider.BatchCallContext(ctx, batch); err != nil {
		return nil, err
	}

	for i, v := range batch {
		if v.Error != nil {
			responses[i].Error = v.Error
		}
	}

	return responses, nil
}

func BatchCheckFileFinalized(provider interfaces.Provider, ctx context.Context, txSeqs ...uint64) ([]BatchRpcResponse[*bool], error) {
	requests := make([]BatchRpcRequest, 0, len(txSeqs))
	for _, v := range txSeqs {
		requests = append(requests, BatchRpcRequest{
			Method: "zgs_checkFileFinalized",
			Args:   []any{v},
		})
	}

	return BatchCallRpc[*bool](provider, ctx, requests...)
}
