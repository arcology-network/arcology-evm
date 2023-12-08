package hashdb

import (
	parahashdb "github.com/arcology-network/concurrent-evm/trie/exp"
)

// "github.com/ethereum/go-ethereum/ethdb"

//	func NewParallelDatabase(diskdbs [16]ethdb.Database, config *Config) *Database {
//		dbs := NewDatabase(diskdbs[0], config) // For preimage
//		dbs.backend = parahashdb.New(diskdbs, config, mptResolver{})
//		return dbs
//	}
// func GetBackendDB(this *Database) *parahashdb.Database {
// 	if db, ok := this.backend.(*parahashdb.Database); ok {
// 		return db
// 	}
// 	return nil
// }

func GetBackendDB() interface{} {
	return parahashdb.Config{}
}
