package hashdb

import (
	"math"
	"sync"
)

func ParallelWorker(total, nThds int, worker func(start, end, idx int, args ...interface{}), args ...interface{}) {
	ranges := make([]int, 0, nThds+1)
	step := int(math.Ceil(float64(total) / float64(nThds)))
	for i := 0; i <= nThds; i++ {
		ranges = append(ranges, int(math.Min(float64(step*i), float64(nThds))))
	}

	var wg sync.WaitGroup
	for i := 0; i < len(ranges)-1; i++ {
		wg.Add(1)
		go func(start int, end int, idx int) {
			defer wg.Done()
			if start != end {
				worker(start, end, idx, args)
			}
		}(ranges[i], ranges[i+1], i)
	}
	wg.Wait()
}
