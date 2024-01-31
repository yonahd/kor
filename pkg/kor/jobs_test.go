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
	})
	job2 := CreateTestJob(testNamespace, "test-job2", &batchv1.JobStatus{
		Succeeded:      1,
		Failed:         0,
		CompletionTime: &v1.Time{Time: time.Now()},
	})

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}

	_, err = clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake job: %v", err)
	}
	return clientset
}

func TestProcessNamespaceJobs(t *testing.T) {
	clientset := createTestJobs(t)

	completedJobs, err := ProcessNamespaceJobs(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(completedJobs) != 1 {
		t.Errorf("Expected 1 job been completed, got %d", len(completedJobs))
	}

	if completedJobs[0] != "test-job2" {
		t.Errorf("job2', got %s", completedJobs[0])
	}
}

func TestGetUnusedJobsStructured(t *testing.T) {
	clientset := createTestJobs(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
	}

	output, err := GetUnusedJobs(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedJobsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Jobs": {"test-job2"},
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

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
