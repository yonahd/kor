package utils

import (
	"runtime"
	"sync"
)

func MaxParallelWorkers() int {
	maxWorkers := runtime.GOMAXPROCS(0)
	if maxWorkers < 1 {
		return 1
	}
	return maxWorkers
}

func RunJobs[T any](jobs []func() T, maxWorkers int, onPanic func(any)) []T {
	if len(jobs) == 0 {
		return nil
	}

	if maxWorkers < 1 {
		maxWorkers = 1
	}

	results := make([]T, len(jobs))
	workers := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, job := range jobs {
		workers <- struct{}{}

		wg.Add(1)
		go func(index int, jobFn func() T) {
			defer wg.Done()
			defer func() {
				<-workers
			}()
			defer func() {
				if recovered := recover(); recovered != nil {
					if onPanic != nil {
						onPanic(recovered)
					}
				}
			}()

			results[index] = jobFn()
		}(i, job)
	}

	wg.Wait()
	return results
}
