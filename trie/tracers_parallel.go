package trie

import (
	"github.com/ethereum/go-ethereum/trie/trienode"
)

type AccessListCache struct {
	tx   uint32
	keys [][]byte
	data [][]byte
}

func NewAccessListCaches(num int) []*AccessListCache {
	list := make([]*AccessListCache, num)
	for i := 0; i < len(list); i++ {
		list[i] = &AccessListCache{
			keys: [][]byte{},
			data: [][]byte{},
		}
	}
	return list
}

func (this *AccessListCache) Add(key []byte, val []byte) {
	this.keys = append(this.keys, key)
	this.data = append(this.data, val)
}

func (this *AccessListCache) Merge(accesses ...*AccessListCache) {
	for _, v := range accesses {
		this.keys = append(this.keys, v.keys...)
		this.data = append(this.data, v.data...)
	}
}

func (this *AccessListCache) ToMap() map[string][]byte {
	hashmap := map[string][]byte{}
	for i, k := range this.keys {
		hashmap[string(k)] = this.data[i]
	}
	return hashmap
}

func (this *AccessListCache) Unique() ([]string, [][]byte) {
	hashmap := this.ToMap()
	keys, values := make([]string, 0, len(hashmap)), make([][]byte, 0, len(hashmap))
	for k, v := range hashmap {
		keys = append(keys, k)
		values = append(values, v)
	}
	return keys, values
}

type parallelTracer struct {
	tracers [17]*tracer
}

// newTracer initializes the parallelTracer for capturing trie changes.
func newParaTracer() tracerInterface {
	paraTracer := &parallelTracer{}
	for i := 0; i < len(paraTracer.tracers); i++ {
		paraTracer.tracers[i] = newTracer()
	}
	return paraTracer
}

// onRead tracks the newly loaded trie node and caches the rlp-encoded
// blob internally. Don't change the value outside of function since
// it's not deep-copied.
func (t *parallelTracer) onRead(path []byte, val []byte) {
	if len(path) > 0 {
		t.tracers[(path[0])].onRead(path, val)
		return
	}
	t.tracers[16].onRead(path, val)
}

// onInsert tracks the newly inserted trie node. If it's already
// in the deletion set (resurrected node), then just wipe it from
// the deletion set as it's "untouched".
func (t *parallelTracer) onInsert(path []byte) {
	if len(path) > 0 {
		t.tracers[path[0]].onInsert(path)
		return
	}
	t.tracers[16].onInsert(path)
}

// onDelete tracks the newly deleted trie node. If it's already
// in the addition set, then just wipe it from the addition set
// as it's untouched.
func (t *parallelTracer) onDelete(path []byte) {
	if len(path) > 0 {
		t.tracers[path[0]].onDelete(path)
		return
	}
	t.tracers[16].onDelete(path)
}

// reset clears the content tracked by parallelTracer.
func (t *parallelTracer) reset() {
	for i := 0; i < len(t.tracers); i++ {
		t.tracers[i].reset()
	}
}

// copy returns a deep copied parallelTracer instance.
func (t *parallelTracer) copy() tracerInterface {
	paraTracer := newParaTracer().(*parallelTracer) //.(*parallelTracer)
	for i := 0; i < len(t.tracers); i++ {
		paraTracer.tracers[i] = t.tracers[i].copy().(*tracer)
	}
	return paraTracer
}

// markDeletions puts all tracked deletions into the provided nodeset.
func (t *parallelTracer) markDeletions(set *trienode.NodeSet) {
	for i := 0; i < len(t.tracers); i++ {
		t.tracers[i].markDeletions(set)
	}
}

// markDeletions puts all tracked deletions into the provided nodeset.
func (t *parallelTracer) getAccessList() map[string][]byte {
	accessList := map[string][]byte{}
	// for i := 0; i < len(t.tracers); i++ {
	// 	for k, v := range t.tracers[i].accessList {
	// 		accessList[k] = v
	// 	}
	// }

	return accessList
}

func (t *parallelTracer) getDeletes() map[string]struct{} {
	deletes := map[string]struct{}{}
	for i := 0; i < len(t.tracers); i++ {
		for k, v := range t.tracers[i].deletes {
			deletes[k] = v
		}
	}
	return deletes
}

func (t *parallelTracer) getInserts() map[string]struct{} {
	inserts := map[string]struct{}{}
	for i := 0; i < len(t.tracers); i++ {
		for k, v := range t.tracers[i].inserts {
			inserts[k] = v
		}
	}
	return inserts
}

func (t *parallelTracer) deletedNodes() []string {
	var paths []string
	for i := 0; i < len(t.tracers); i++ {
		for path := range t.tracers[i].deletes {
			paths = append(paths, path)
		}
	}
	return paths
}
