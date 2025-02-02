package hashdb

import (
	"errors"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
)

type Database struct {
	dbs [16]*database
}

// diskdbs, db.cleans, mptResolver{}
func New(diskdb interface{}, _ interface{}, resolver ChildResolver, config *Config) *Database {
	db := &Database{}
	if ddb, ok := diskdb.(ethdb.Database); ok {
		for i := 0; i < len(db.dbs); i++ {
			db.dbs[i] = new(ddb, config, resolver) // rawdb.NewMemoryDatabase()
		}
		return db
	}

	if hdbs, ok := diskdb.([16]ethdb.Database); ok {
		for i := 0; i < len(hdbs); i++ {
			db.dbs[i] = new(hdbs[i], config, resolver) // rawdb.NewMemoryDatabase()
		}
		return db
	}
	return nil
}

func NewWithCache(diskdb interface{}, _ interface{}, resolver ChildResolver, sharedCleanCache *fastcache.Cache, config *Config) *Database {
	db := &Database{}
	if ddb, ok := diskdb.(ethdb.Database); ok {
		for i := 0; i < len(db.dbs); i++ {
			db.dbs[i] = newWithSharedCache(ddb, config, resolver, sharedCleanCache) // rawdb.NewMemoryDatabase()
		}
		return db
	}

	if hdbs, ok := diskdb.([16]ethdb.Database); ok {
		for i := 0; i < len(hdbs); i++ {
			db.dbs[i] = newWithSharedCache(hdbs[i], config, resolver, sharedCleanCache) // rawdb.NewMemoryDatabase()
		}
		return db
	}

	return nil
}

func (this *Database) DBs() [16]ethdb.Database {
	dbs := [16]ethdb.Database{}
	for i := range this.dbs {
		dbs[i] = this.dbs[i].diskdb
	}
	return dbs
}

func (this *Database) Find(node common.Hash) (*database, []byte, error) {
	for i := 0; i < len(this.dbs); i++ {
		if b, err := this.dbs[i].Node(node); err == nil && len(b) > 0 {
			return this.dbs[i], b, nil
		}
	}
	return nil, nil, errors.New("Node not found!")
}

func (this *Database) shard(hash []byte) *database { return this.dbs[hash[0]>>4] }
func (this *Database) Scheme() string              { return rawdb.HashScheme }
func (this *Database) Reader(blockRoot common.Hash) (*paraReader, error) {
	return &paraReader{this}, nil
}
func (this *Database) Node(hash common.Hash) ([]byte, error) {
	return this.shard(hash[:]).Node(hash)
}

func (this *Database) Reference(root common.Hash, parent common.Hash) {
	this.shard(parent[:]).Reference(root, parent)
}

func (this *Database) Dereference(root common.Hash) {
	this.shard(root[:]).Dereference(root)
}

type paraReader struct {
	dbs *Database
}

func (this *paraReader) Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	if len(path) > 0 {
		return this.dbs.dbs[path[0]].Node(hash)
	}
	return this.dbs.shard(hash[:]).Node(hash)
}

func (this *Database) Initialized(genesisRoot common.Hash) bool {
	return rawdb.HasLegacyTrieNode(this.dbs[0].diskdb, genesisRoot)
}

func (this *Database) Size() (common.StorageSize, common.StorageSize) {
	total := common.StorageSize(0)
	for i := 0; i < len(this.dbs); i++ {
		this.dbs[i].lock.Lock()
		_, size := this.dbs[i].Size()
		total += size
		this.dbs[i].lock.Unlock()
	}
	return 0, total
}

func (this *Database) Update(root common.Hash, parent common.Hash, block uint64, nodes *trienode.MergedNodeSet, states *triestate.Set) error {
	if parent != types.EmptyRootHash {
		if blob, _ := this.shard(parent[:]).Node(parent); len(blob) == 0 {
			log.Error("parent state is not present")
		}
	}

	sharded, rootShard, _ := nodes.Regroup()

	updater := func(start, end, _ int, _ ...interface{}) {
		this.dbs[start].Update(root, common.Hash{}, block, sharded[start], states)
	}
	ParallelWorker(len(sharded), len(sharded), updater)

	// this.dbs[node[0]>>4].Commit(node, report)
	this.shard(root[:]).Update(root, common.Hash{}, block, rootShard, states)
	return nil
}

func (this *Database) Commit(hash common.Hash, report bool) error {
	encodedNode, err := this.shard(hash[:]).Node(hash)
	if err != nil {
		return err
	}

	children := []common.Hash{hash}
	this.shard(hash[:]).resolver.ForEach(encodedNode, func(child common.Hash) {
		children = append(children, child)
	})

	for i := 0; i < len(children); i++ {
		if shard, _, err := this.Find(children[i]); shard != nil {
			if err := shard.Commit(children[i], report); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

func (this *Database) Close() error {
	for i := 0; i < len(this.dbs); i++ {
		if err := this.dbs[i].Close(); err != nil {
			return err
		}
	}
	return nil
}

func (this *Database) Cap(limit common.StorageSize) error {
	for i := 0; i < len(this.dbs); i++ {
		if err := this.dbs[i].Cap(limit / common.StorageSize(len(this.dbs))); err != nil {
			return err
		}
	}
	return nil
}
