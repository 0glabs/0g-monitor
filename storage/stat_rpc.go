package storage

import (
	"context"
	"time"

	"github.com/0glabs/0g-storage-client/node"
	"github.com/Conflux-Chain/go-conflux-util/parallel"
	providers "github.com/openweb3/go-rpc-provider/provider_wrapper"
	"github.com/sirupsen/logrus"
)

type QueryRpcResult[T any] struct {
	Data    T
	Err     error
	Latency time.Duration
}

type rpcExecutor[T any] struct {
	nodes        []string
	rpcFunc      func(*node.ZgsClient, context.Context) (T, error)
	node2Results map[string]*QueryRpcResult[T]

	lastReportTime time.Time
}

func ParallelQueryRpc[T any](ctx context.Context, nodes []string, rpcFunc func(*node.ZgsClient, context.Context) (T, error), opt ...parallel.SerialOption) (map[string]*QueryRpcResult[T], error) {
	executor := rpcExecutor[T]{
		nodes:          nodes,
		rpcFunc:        rpcFunc,
		node2Results:   make(map[string]*QueryRpcResult[T]),
		lastReportTime: time.Now(),
	}

	if err := parallel.Serial(ctx, &executor, len(nodes), opt...); err != nil {
		return nil, err
	}

	return executor.node2Results, nil
}

// ParallelDo implements the parallel.Interface interface.
func (executor *rpcExecutor[T]) ParallelDo(ctx context.Context, routine, task int) (*QueryRpcResult[T], error) {
	client, err := node.NewZgsClient(executor.nodes[task], providers.Option{RequestTimeout: 3 * time.Second})
	if err != nil {
		return &QueryRpcResult[T]{Err: err}, nil
	}
	defer client.Close()

	var result QueryRpcResult[T]
	start := time.Now()
	result.Data, result.Err = executor.rpcFunc(client, ctx)
	result.Latency = time.Since(start)

	return &result, nil
}

// ParallelCollect implements the parallel.Interface interface.
func (executor *rpcExecutor[T]) ParallelCollect(ctx context.Context, result *parallel.Result[*QueryRpcResult[T]]) error {
	node := executor.nodes[result.Task]
	executor.node2Results[node] = result.Value

	if time.Since(executor.lastReportTime) > 5*time.Second {
		logrus.WithFields(logrus.Fields{
			"total":     len(executor.nodes),
			"completed": result.Task,
		}).Info("Progress update")

		executor.lastReportTime = time.Now()
	}

	return nil
}
