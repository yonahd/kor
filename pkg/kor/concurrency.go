package kor

import (
	"fmt"
	"os"
	"runtime"
	"sync"
)

func getMaxParallelResourceWorkers() int {
	maxWorkers := runtime.GOMAXPROCS(0)
	if maxWorkers < 1 {
		return 1
	}
	return maxWorkers
}

func runResourceDiffJobs(jobs []func() ResourceDiff, maxWorkers int) []ResourceDiff {
	if len(jobs) == 0 {
		return nil
	}

	if maxWorkers < 1 {
		maxWorkers = 1
	}

	results := make([]ResourceDiff, len(jobs))
	workers := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, job := range jobs {
		workers <- struct{}{}

		wg.Add(1)
		go func(index int, diffFn func() ResourceDiff) {
			defer wg.Done()
			defer func() {
				<-workers
			}()
			defer func() {
				if recovered := recover(); recovered != nil {
					fmt.Fprintf(os.Stderr, "Failed to process resource diff job: %v\n", recovered)
				}
			}()

			results[index] = diffFn()
		}(i, job)
	}

	wg.Wait()
	return results
}
