package trie

import (
	"github.com/ethereum/go-ethereum/trie/trienode"
)

type tracerInterface interface {
	onRead([]byte, []byte)
	onInsert([]byte)
	onDelete(path []byte)
	reset()
	copy() tracerInterface
	markDeletions(set *trienode.NodeSet)
	getAccessList() map[string][]byte
	getDeletes() map[string]struct{}
	getInserts() map[string]struct{}
	deletedNodes() []string
}
