package files

import "github.com/0glabs/0g-storage-client/common/shard"

type ShardCounter struct {
	shard2Id2Counts map[uint64]map[uint64]int
}

func NewShardCounter() *ShardCounter {
	return &ShardCounter{
		shard2Id2Counts: make(map[uint64]map[uint64]int),
	}
}

func (counter *ShardCounter) Insert(config shard.ShardConfig) {
	if id2Counts, ok := counter.shard2Id2Counts[config.NumShard]; ok {
		id2Counts[config.ShardId]++
	} else {
		counter.shard2Id2Counts[config.NumShard] = map[uint64]int{
			config.ShardId: 1,
		}
	}
}

func (counter *ShardCounter) Replica() int {
	var replica int

	for numShard, id2Counts := range counter.shard2Id2Counts {
		// any shard id missded
		if uint64(len(id2Counts)) < numShard {
			continue
		}

		min := id2Counts[0]

		for _, count := range id2Counts {
			if min > count {
				min = count
			}
		}

		replica += min
	}

	return replica
}

func (counter *ShardCounter) Items() map[uint64]map[uint64]int {
	return counter.shard2Id2Counts
}
