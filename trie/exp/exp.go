package hashdb
package pathdb

type Config struct {
	StateHistory   uint64 // Number of recent blocks to maintain state history for
	CleanCacheSize int    // Maximum memory allowance (in bytes) for caching clean nodes
	DirtyCacheSize int    // Maximum memory allowance (in bytes) for caching dirty nodes
	ReadOnly       bool   // Flag whether the database is opened in read only mode.
}
