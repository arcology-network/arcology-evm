package trie

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/ethdb"
	hashdb "github.com/ethereum/go-ethereum/trie/triedb/parahashdb"
	parahashdb "github.com/ethereum/go-ethereum/trie/triedb/parahashdb"
)

func NewParallelDatabase(diskdbs [16]ethdb.Database, config *Config) *Database {
	dbs := NewDatabase(diskdbs[0], config) // For preimage

	dbConfig := &hashdb.Config{CleanCacheSize: 1024 * 1024 * 10}
	dbs.backend = parahashdb.New(diskdbs, config, mptResolver{}, dbConfig)
	return dbs
}

func NewParallelDatabaseWithSharedCache(diskdbs [16]ethdb.Database, cleanCache *fastcache.Cache, config *Config) *Database {
	dbs := NewDatabase(diskdbs[0], config) // For preimage
	dbs.backend = parahashdb.NewWithCache(diskdbs, config, mptResolver{}, cleanCache, nil)
	return dbs
}

func GetBackendDB(this *Database) *parahashdb.Database {
	if db, ok := this.backend.(*parahashdb.Database); ok {
		return db
	}
	return nil
}
