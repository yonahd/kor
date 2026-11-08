package utils

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestRunJobsReturnsAllResults(t *testing.T) {
	jobs := []func() string{
		func() string { return "ConfigMap" },
		func() string { return "Service" },
		func() string { return "Secret" },
	}

	results := RunJobs(jobs, 2, nil)
	if len(results) != len(jobs) {
		t.Fatalf("Expected %d results, got %d", len(jobs), len(results))
	}

	if results[0] != "ConfigMap" {
		t.Fatalf("Unexpected first result: %q", results[0])
	}
	if results[1] != "Service" {
		t.Fatalf("Unexpected second result: %q", results[1])
	}
	if results[2] != "Secret" {
		t.Fatalf("Unexpected third result: %q", results[2])
	}
}

func TestRunJobsImprovesSpeedWithParallelism(t *testing.T) {
	sleepDuration := 40 * time.Millisecond
	jobs := []func() string{
		func() string { time.Sleep(sleepDuration); return "ConfigMap" },
		func() string { time.Sleep(sleepDuration); return "Service" },
		func() string { time.Sleep(sleepDuration); return "Secret" },
		func() string { time.Sleep(sleepDuration); return "Pod" },
	}

	start := time.Now()
	RunJobs(jobs, 1, nil)
	sequentialDuration := time.Since(start)

	start = time.Now()
	RunJobs(jobs, len(jobs), nil)
	parallelDuration := time.Since(start)
	t.Logf("sequential duration=%s parallel duration=%s", sequentialDuration, parallelDuration)

	if parallelDuration >= sequentialDuration {
		t.Fatalf("Expected parallel execution to be faster. sequential=%s parallel=%s", sequentialDuration, parallelDuration)
	}
}

func TestRunJobsLimitsSpawnedGoroutines(t *testing.T) {
	const (
		jobCount       = 1000
		maxWorkers     = 1
		maxGoroutines  = 64
		settleDuration = 50 * time.Millisecond
	)

	baseline := runtime.NumGoroutine()
	firstJobStarted := make(chan struct{})
	releaseJobs := make(chan struct{})
	var startedOnce sync.Once

	jobs := make([]func() string, 0, jobCount)
	for i := 0; i < jobCount; i++ {
		jobs = append(jobs, func() string {
			startedOnce.Do(func() {
				close(firstJobStarted)
			})
			<-releaseJobs
			return "ConfigMap"
		})
	}

	done := make(chan struct{})
	go func() {
		RunJobs(jobs, maxWorkers, nil)
		close(done)
	}()

	select {
	case <-firstJobStarted:
	case <-time.After(2 * time.Second):
		t.Fatalf("Timed out waiting for the first job to start")
	}

	time.Sleep(settleDuration)

	current := runtime.NumGoroutine()
	if current-baseline > maxGoroutines {
		t.Fatalf("Expected bounded goroutine usage, got baseline=%d current=%d delta=%d", baseline, current, current-baseline)
	}

	close(releaseJobs)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Timed out waiting for RunJobs to complete")
	}
}

func TestRunJobsRecoversFromPanics(t *testing.T) {
	var panicked any
	jobs := []func() string{
		func() string {
			panic("boom")
		},
		func() string {
			return "Service"
		},
	}

	results := RunJobs(jobs, 2, func(recovered any) {
		panicked = recovered
	})

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results[0] != "" {
		t.Fatalf("Expected zero-value result for panicked job, got %q", results[0])
	}

	if results[1] != "Service" {
		t.Fatalf("Unexpected second result: %q", results[1])
	}

	if panicked != "boom" {
		t.Fatalf("Expected panic handler to receive %q, got %#v", "boom", panicked)
	}
}

func TestMaxParallelWorkers(t *testing.T) {
	workers := MaxParallelWorkers()
	if workers < 1 {
		t.Fatalf("Expected at least 1 worker, got %d", workers)
	}
}
