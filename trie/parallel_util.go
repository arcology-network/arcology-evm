package trie

import (
	"math"
	"sort"
	"sync"
)

func GenerateRanges(length int, numThreads int) []int {
	ranges := make([]int, 0, numThreads+1)
	step := int(math.Ceil(float64(length) / float64(numThreads)))
	for i := 0; i <= numThreads; i++ {
		ranges = append(ranges, int(math.Min(float64(step*i), float64(length))))
	}
	return ranges
}

func ParallelWorker(total, nThds int, worker func(start, end, idx int, args ...interface{}), args ...interface{}) {
	idxRanges := GenerateRanges(total, nThds)
	var wg sync.WaitGroup
	for i := 0; i < len(idxRanges)-1; i++ {
		wg.Add(1)
		go func(start int, end int, idx int) {
			defer wg.Done()
			if start != end {
				worker(start, end, idx, args)
			}
		}(idxRanges[i], idxRanges[i+1], i)
	}
	wg.Wait()
}

// func ParallelWorker(total, nThds int, worker func(start, end, idx int, args ...interface{}), args ...interface{}) {
// 	ranges := make([]int, 0, nThds+1)
// 	step := int(math.Ceil(float64(total) / float64(nThds)))
// 	for i := 0; i <= nThds; i++ {
// 		ranges = append(ranges, int(math.Min(float64(step*i), float64(nThds))))
// 	}

// 	var wg sync.WaitGroup
// 	for i := 0; i < len(ranges)-1; i++ {
// 		wg.Add(1)
// 		go func(start int, end int, idx int) {
// 			defer wg.Done()
// 			if start != end {
// 				worker(start, end, idx, args)
// 			}
// 		}(ranges[i], ranges[i+1], i)
// 	}
// 	wg.Wait()
// }

func SortBy1st[T0 any, T1 any](first []T0, second []T1, compare func(T0, T0) bool) {
	array := make([]struct {
		_0 T0
		_1 T1
	}, len(first))

	for i := range array {
		array[i]._0 = first[i]
		array[i]._1 = second[i]
	}
	sort.SliceStable(array, func(i, j int) bool { return compare(array[i]._0, array[j]._0) })

	for i := range array {
		first[i] = array[i]._0
		second[i] = array[i]._1
	}
}
