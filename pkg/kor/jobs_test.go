package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestJobs(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	job1 := CreateTestJob(testNamespace, "test-job1", &batchv1.JobStatus{
		Succeeded: 0,
		Failed:    1,
	}, AppLabels)

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	job2 := CreateTestJob(testNamespace, "test-job2", &batchv1.JobStatus{
		Succeeded:      1,
		Failed:         0,
		CompletionTime: &v1.Time{Time: time.Now()},
	}, AppLabels)

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	job3 := CreateTestJob(testNamespace, "test-job3", &batchv1.JobStatus{
		Succeeded: 0,
		Failed:    1,
	}, UsedLabels)

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	job4 := CreateTestJob(testNamespace, "test-job4", &batchv1.JobStatus{
		Succeeded:      1,
		Failed:         0,
		CompletionTime: &v1.Time{Time: time.Now()},
	}, UnusedLabels)

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	job5 := CreateTestJob(testNamespace, "test-job5", &batchv1.JobStatus{
		Succeeded: 0,
		Failed:    1,
		Conditions: []batchv1.JobCondition{
			{
				Type:    batchv1.JobFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "BackoffLimitExceeded",
				Message: "Job has reached the specified backoff limit",
			},
		},
	}, AppLabels)

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job5, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	job6 := CreateTestJob(testNamespace, "test-job6", &batchv1.JobStatus{
		Succeeded: 0,
		Failed:    1,
		Conditions: []batchv1.JobCondition{
			{
				Type:    batchv1.JobFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "DeadlineExceeded",
				Message: "Job was active longer than specified deadline",
			},
		},
	}, AppLabels)

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job6, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	job7 := CreateTestJob(testNamespace, "test-job7", &batchv1.JobStatus{
		Succeeded: 0,
		Failed:    1,
		Conditions: []batchv1.JobCondition{
			{
				Type:    batchv1.JobFailed,
				Status:  corev1.ConditionTrue,
				Reason:  "FailedIndexes",
				Message: "Job has failed indexes",
			},
		},
	}, AppLabels)

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job7, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	job8 := CreateTestJob(testNamespace, "test-job8", &batchv1.JobStatus{
		Succeeded: 0,
		Failed:    1,
		Conditions: []batchv1.JobCondition{
			{
				Type:    batchv1.JobSuspended,
				Status:  corev1.ConditionTrue,
				Reason:  "JobSuspended",
				Message: "Job suspended",
			},
		},
	}, AppLabels)

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job8, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	return clientset
}

func TestProcessNamespaceJobs(t *testing.T) {
	clientset := createTestJobs(t)

	unusedJobs, err := processNamespaceJobs(clientset, testNamespace, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedJobsNames := []string{"test-job2", "test-job4", "test-job5", "test-job6", "test-job7", "test-job8"}

	if len(unusedJobs) != len(expectedJobsNames) {
		t.Errorf("Expected %d jobs unused got %d", len(expectedJobsNames), len(unusedJobs))
	}

	for i, job := range unusedJobs {
		if job.Name != expectedJobsNames[i] {
			t.Errorf("Expected %s, got %s", expectedJobsNames[i], job.Name)
		}
	}
}

func TestGetUnusedJobsStructured(t *testing.T) {
	clientset := createTestJobs(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedJobs(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedJobsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Job": {
				"test-job2",
				"test-job4",
				"test-job5",
				"test-job6",
				"test-job7",
				"test-job8",
			},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestFilterCronJobOwnedJobs(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	// Create two jobs - one owned by cronjob, one standalone
	// Job owned by cronjob (completed)
	ownedJob := CreateTestJob(testNamespace, "cronjob-owned-job", &batchv1.JobStatus{
		Succeeded:      1,
		Failed:         0,
		CompletionTime: &v1.Time{Time: time.Now()},
	}, AppLabels)
	// Add owner reference to cronjob
	ownedJob.OwnerReferences = []v1.OwnerReference{
		{
			Kind: "CronJob",
			Name: "test-cronjob",
		},
	}
	
	// Standalone Job (completed)
	standaloneJob := CreateTestJob(testNamespace, "standalone-job", &batchv1.JobStatus{
		Succeeded:      1,
		Failed:         0,
		CompletionTime: &v1.Time{Time: time.Now()},
	}, AppLabels)

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), ownedJob, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), standaloneJob, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	// Test without filter - should return both (both are completed)
	filterOptsNoSkip := &filters.Options{IgnoreOwnerReferences: false}
	unusedWithoutFilter, err := processNamespaceJobs(clientset, testNamespace, filterOptsNoSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused jobs: %v", err)
	}

	if len(unusedWithoutFilter) != 2 {
		t.Errorf("Expected 2 unused Job objects without filter, got %d", len(unusedWithoutFilter))
	}

	// Test with filter - should return only standalone
	filterOptsWithSkip := &filters.Options{IgnoreOwnerReferences: true}
	unusedWithFilter, err := processNamespaceJobs(clientset, testNamespace, filterOptsWithSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused jobs: %v", err)
	}

	if len(unusedWithFilter) != 1 {
		t.Errorf("Expected 1 unused Job object with filter, got %d", len(unusedWithFilter))
	}

	if unusedWithFilter[0].Name != "standalone-job" {
		t.Errorf("Expected standalone-job to be unused, got %s", unusedWithFilter[0].Name)
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
