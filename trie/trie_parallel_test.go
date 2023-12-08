package trie

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
)

func TestAccessListCache(t *testing.T) {
	list := NewAccessListCaches(2)

	list[0].keys = [][]byte{{1}, {2}, {7}}
	list[0].data = [][]byte{{11}, {21}, {77}}

	list[1].keys = [][]byte{{3}, {4}, {7}}
	list[1].data = [][]byte{{21}, {22}, {77}}

	target := &AccessListCache{
		tx:   1,
		keys: [][]byte{},
		data: [][]byte{},
	}

	target.Merge(list...)
	k, v := target.Unique()

	SortBy1st(k, v, func(_0, _1 string) bool {
		return _0 < _1
	})

	if !reflect.DeepEqual(k, []string{string([]byte{1}), string([]byte{2}), string([]byte{3}), string([]byte{4}), string([]byte{7})}) {
		t.Error("Failed compare")
	}

	if !reflect.DeepEqual(v, [][]byte{{11}, {21}, {21}, {22}, {77}}) {
		t.Error("Failed compare")
	}
}

func TestParallelUpdateionPutSmallDataSet(t *testing.T) {
	keys := make([][]byte, 2)

	keys[0], keys[1] = make([]byte, 20), make([]byte, 20)
	for i := 0; i < len(keys[0]); i++ {
		keys[0][i] = 'a'
		keys[1][i] = 'b'
	}

	paraDB := NewParallelDatabase(new16TestMemDBs(), nil)
	paraTrie16 := NewEmptyParallel(paraDB)

	paraTrie16.ParallelUpdate(keys, keys)
	paraTrie16Root, paraNodes, err := paraTrie16.Commit(false)
	if err != nil {
		t.Error(err)
	}

	paraDB.Update(paraTrie16Root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(paraNodes), &triestate.Set{})

	for i, k := range keys {
		if v, err := paraTrie16.Get(k); err != nil {
			t.Error(err)
		} else {
			if !bytes.Equal(v, keys[i]) {
				t.Error("Mismatch")
			}
		}
	}

	newParaTrie, _ := New(TrieID(paraTrie16Root), paraDB)
	output, _ := newParaTrie.ParallelGet(keys)
	for i := 0; i < len(keys); i++ {
		if !bytes.Equal(output[i], keys[i]) {
			t.Errorf("Wrong value")
		}
	}

	for _, k := range keys {
		proofs := memorydb.New()
		newParaTrie.Prove(k, proofs)

		v, err := VerifyProof(newParaTrie.Hash(), k, proofs)
		if len(v) == 0 || err != nil || !bytes.Equal(v, k) {
			t.Errorf("Wrong Proof")
		}
	}
}

func TestParallelUpdateionPut(t *testing.T) {
	keys := make([][]byte, 122)
	data := make([][]byte, len(keys))
	for i := 0; i < len(data); i++ {
		keys[i] = crypto.Keccak256([]byte(fmt.Sprint(i)))
		data[i] = []byte(fmt.Sprint(i))
	}

	paraDB := NewParallelDatabase(new16TestMemDBs(), nil)
	paraTrie16 := NewEmptyParallel(paraDB)

	paraTrie16.ParallelUpdate(keys, data)
	paraTrie16Root, paraNodes, err := paraTrie16.Commit(false)
	if err != nil {
		t.Error(err)
	}

	paraDB.Update(paraTrie16Root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(paraNodes), &triestate.Set{})

	for i, k := range keys {
		if v, err := paraTrie16.Get(k); err != nil {
			t.Error(err)
		} else {
			if !bytes.Equal(v, data[i]) {
				t.Error("Mismatch")
			}
		}
	}

	newParaTrie, _ := New(TrieID(paraTrie16Root), paraDB)
	output, _ := newParaTrie.ParallelGet(keys)
	for i := 0; i < len(data); i++ {
		if !bytes.Equal(output[i], data[i]) {
			t.Errorf("Wrong value")
		}
	}

	for i, k := range keys {
		proofs := memorydb.New()
		newParaTrie.Prove(k, proofs)

		v, err := VerifyProof(newParaTrie.Hash(), k, proofs)
		if len(v) == 0 || err != nil || !bytes.Equal(v, data[i]) {
			t.Errorf("Wrong Proof")
		}
	}
}

func TestParallelUpdateionConsistency(t *testing.T) {
	keys := make([][]byte, 122)
	data := make([][]byte, len(keys))
	for i := 0; i < len(data); i++ {
		keys[i] = crypto.Keccak256([]byte(fmt.Sprint(i)))
		data[i] = crypto.Keccak256([]byte(fmt.Sprint(i)))
	}

	fmt.Println(len(keybytesToHex(keys[0])))

	db := NewDatabase(rawdb.NewMemoryDatabase(), HashDefaults)
	trie := NewEmpty(db)
	for i, k := range keys {
		trie.MustUpdate(k, data[i])
	}

	serialRoot := trie.Hash()
	// ==================== Parallel trie ====================
	paraDB := NewParallelDatabase(new16TestMemDBs(), nil)
	paraTrie16 := NewEmptyParallel(paraDB)
	// ParallelTask{}.Insert(paraTrie16, keys, data)
	paraTrie16.ParallelUpdate(keys, data)
	paraTrie16.ParallelUpdate(keys, data) // Insert twice
	paraTrie16Root, paraNodes, err := paraTrie16.Commit(false)
	if err != nil {
		t.Error(err)
	}

	paraDB.Update(paraTrie16Root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(paraNodes), &triestate.Set{})
	// paraTrie16Root := paraTrie16.Hash()

	newParaTrie, err := New(TrieID(paraTrie16Root), paraDB)
	if err != nil {
		t.Error("Failed to open the DB")
	}

	fmt.Println("Sequence put: ", serialRoot)
	fmt.Println("Parallel put: ", paraTrie16Root)

	if serialRoot != paraTrie16Root {
		t.Errorf("expected %x got %x", serialRoot, paraTrie16Root)
	}

	for i, k := range keys {
		if v, err := newParaTrie.Get(k); err != nil {
			t.Error(err)
		} else {
			if !bytes.Equal(v, data[i]) {
				t.Error("Mismatch")
			}
		}
	}

	root, nodes, err := trie.Commit(true)
	if err != nil {
		t.Error(err)
	}

	db.Update(root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(nodes), &triestate.Set{})

	newTrie, err := New(TrieID(root), db)
	if err != nil {
		t.Error(err)
	}

	for i, k := range keys {
		if v, err := newTrie.Get(k); err != nil {
			t.Error(err)
		} else {
			if !bytes.Equal(v, data[i]) {
				t.Error("Mismatch")
			}
		}
	}

	for i, k := range keys {
		if v, err := newParaTrie.Get(k); err != nil {
			t.Error(err)
		} else {
			if !bytes.Equal(v, data[i]) {
				t.Error("Mismatch")
			}
		}
	}

	output, _ := trie.ParallelGet(keys)
	for i := 0; i < len(data); i++ {
		if !bytes.Equal(output[i], data[i]) {
			t.Errorf("Wrong value")
		}
	}
}

func TestRace(t *testing.T) {
	keys := make([][]byte, 1000)
	data := make([][]byte, len(keys))
	for i := 0; i < len(data); i++ {
		keys[i] = crypto.Keccak256([]byte(fmt.Sprint(i)))
		data[i] = crypto.Keccak256([]byte(fmt.Sprint(i + len(keys))))
	}

	trie := NewEmptyParallel(NewParallelDatabase(new16TestMemDBs(), nil))
	trie.ParallelUpdate(keys, data)

	ParallelWorker(len(keys), 8, func(start, end, _ int, _ ...interface{}) {
		for i := start; i < end; i++ {
			if v, _ := trie.Get(keys[i]); !bytes.Equal(v, data[i]) {
				t.Error("Mismatch values")
			}
		}
	})
}

func TestParallelTrieGet(t *testing.T) {
	keys := make([][]byte, 1000000)
	data := make([][]byte, len(keys))
	for i := 0; i < len(data); i++ {
		keys[i] = crypto.Keccak256([]byte(fmt.Sprint(i)))
		data[i] = crypto.Keccak256([]byte(fmt.Sprint(i + len(keys))))
	}

	db := NewDatabase(rawdb.NewMemoryDatabase(), HashDefaults)
	trie := NewEmpty(db)
	for i, k := range keys {
		trie.MustUpdate(k, data[i])
	}

	t0 := time.Now()
	for i, k := range keys {
		v, err := trie.Get(k)
		if !bytes.Equal(v, data[i]) {
			fmt.Println(err)
		}
	}
	fmt.Println("Get ", len(keys), time.Since(t0))

	t0 = time.Now()
	ParallelWorker(len(keys), 8, func(start, end, _ int, _ ...interface{}) {
		for i := start; i < end; i++ {
			trie.Get(keys[i])
		}
	})
	fmt.Println("Parallel Get ", len(keys), time.Since(t0))
}

func TestSwitchingTries(t *testing.T) {
	keys := [][]byte{{1, 1, 1}, {2, 2, 2}, {3, 3, 3}, {4, 4, 4}}
	data := keys

	db := NewDatabase(rawdb.NewMemoryDatabase(), HashDefaults)
	trie := NewEmpty(db)
	for i, k := range keys {
		trie.MustUpdate(k, data[i])
	}

	rootNode := trie.root
	root, nodes, err := trie.Commit(false)
	if err != nil {
		t.Error(err)
	}

	db.Update(root, types.EmptyRootHash, 0, trienode.NewWithNodeSet(nodes), &triestate.Set{})
	db.Commit(root, false) // This is optional

	// Reopen a new tir
	newTrie, _ := New(TrieID(root), db)
	for i, k := range keys {
		if v, err := newTrie.Get(k); err != nil {
			t.Error(err)
		} else {
			if !bytes.Equal(v, data[i]) {
				t.Error("Mismatch")
			}
		}
	}

	keys2 := [][]byte{{4, 4, 4}, {3, 3, 3}, {2, 2, 2}, {1, 1, 1}}
	data2 := [][]byte{{4, 4, 4, 4}, {3, 3, 3, 3}, {2, 2, 2, 2}, {1, 1, 1, 1}}

	for i, k := range keys2 {
		newTrie.MustUpdate(k, data2[i])
	}

	for i, k := range keys2 {
		if v, err := newTrie.Get(k); err != nil {
			t.Error(err)
		} else {
			if !bytes.Equal(v, data2[i]) {
				t.Error("Mismatch")
			}
		}
	}
	newTrie.Hash()
	// rootNode := newTrie.root
	// root2, nodes2 := newTrie.Commit(false)

	newTrie2 := newTrie
	// db.Update(root2, types.EmptyRootHash, trienode.NewWithNodeSet(nodes2))
	// newTrie2, _ := New(TrieID(root2), db)

	for i, k := range keys2 {
		if v, err := newTrie2.Get(k); err != nil {
			t.Error(err)
		} else {
			if !bytes.Equal(v, data2[i]) {
				t.Error("Mismatch")
			}
		}
	}

	// newTrie2, _ = New(TrieID(root), db)
	newTrie2.root = rootNode
	for i, k := range keys {
		if v, err := newTrie2.Get(k); err != nil {
			t.Error(err)
		} else {
			if !bytes.Equal(v, data[i]) {
				t.Error("Mismatch")
			}
		}
	}
}

func TestMptPerformance(t *testing.T) {
	trie := NewEmpty(NewDatabase(rawdb.NewMemoryDatabase(), HashDefaults))
	res := trie.Hash()
	exp := types.EmptyRootHash
	if res != exp {
		t.Errorf("expected %x got %x", exp, res)
	}

	keys := make([][]byte, 1000000)
	data := make([][]byte, len(keys))
	for i := 0; i < len(data); i++ {
		keys[i] = crypto.Keccak256([]byte(fmt.Sprint(i)))
		data[i] = crypto.Keccak256([]byte(fmt.Sprint(i)))
	}

	t0 := time.Now()
	for i, k := range keys {
		trie.MustUpdate(k, data[i])
	}
	serialRoot := trie.Hash()
	fmt.Println("Serial put:            "+fmt.Sprint(len(data)), time.Since(t0), serialRoot)

	paraTrie := NewEmptyParallel(NewParallelDatabase(new16TestMemDBs(), nil))

	t0 = time.Now()
	for i, k := range keys {
		paraTrie.MustUpdate(k, data[i])
	}
	paraRoot := paraTrie.Hash()
	fmt.Println("Paral put thread = 1:  "+fmt.Sprint(len(data)), time.Since(t0), paraRoot)

	paraTrie = NewEmptyParallel(NewDatabase(rawdb.NewMemoryDatabase(), HashDefaults))
	t0 = time.Now()
	paraTrie.ParallelUpdate(keys, data)
	// paraRoot = paraTrie.Hash()
	fmt.Println("Paral put thread = 16: "+fmt.Sprint(len(data)), time.Since(t0), paraRoot)

	if serialRoot != paraRoot {
		t.Errorf("expected %x got %x", serialRoot, paraRoot)
	}

	t0 = time.Now()
	for _, k := range keys {
		trie.Get(k)
	}
	fmt.Println("Get ", len(keys), " entries in ", time.Since(t0))

	t0 = time.Now()
	trie.ParallelGet(keys)
	fmt.Println("ParallelGet ", len(keys), " entries in ", time.Since(t0))

	t0 = time.Now()
	trie.ParallelGet(keys)
	fmt.Println("ParallelThreadSafeGet ", len(keys), " entries in ", time.Since(t0))
}
