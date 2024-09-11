package files

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/0glabs/0g-storage-client/contract"
	"github.com/0glabs/0g-storage-client/indexer"
	"github.com/0glabs/0g-storage-client/node"
	"github.com/openweb3/web3go"
	"github.com/pkg/errors"
)

type Sampler struct {
	flow     *contract.FlowCaller
	maxTxSeq atomic.Uint64
	r        *rand.Rand
}

func NewSampler(indexerClient *indexer.Client, w3Client *web3go.Client) (*Sampler, error) {
	nodes, err := indexerClient.GetShardedNodes(context.Background())
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to retrieve trusted nodes from indexer")
	}

	if len(nodes.Trusted) == 0 {
		return nil, errors.New("No trusted nodes retrieved from indexer")
	}

	zgsClient, err := node.NewZgsClient(nodes.Trusted[0].URL, defaultProviderOption)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create zgs client")
	}
	defer zgsClient.Close()

	status, err := zgsClient.GetStatus(context.Background())
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to retrieve status from storage node")
	}
	logger.WithField("flow", status.NetworkIdentity.FlowContractAddress).Info("Succeeded to get status from trusted node")

	caller, _ := w3Client.ToClientForContract()
	flow, err := contract.NewFlowCaller(status.NetworkIdentity.FlowContractAddress, caller)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create flow contract caller")
	}

	s := Sampler{
		flow: flow,
		r:    rand.New(rand.NewSource(time.Now().Unix())),
	}

	if err = s.Update(); err != nil {
		return nil, errors.WithMessage(err, "Failed to initialize sampler")
	}

	return &s, nil
}

func (s *Sampler) Update() error {
	num, err := s.flow.NumSubmissions(nil)
	if err != nil {
		return errors.WithMessage(err, "Failed to retrieve num submissions from flow contract")
	}

	s.maxTxSeq.Store(num.Uint64())

	return nil
}

func (s *Sampler) Random() uint64 {
	maxTxSeq := s.maxTxSeq.Load()
	return uint64(s.r.Int63n(int64(maxTxSeq + 1)))
}
