package kor

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	fake "k8s.io/client-go/kubernetes/fake"
)

func TestDeleteResource(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	configmap1 := CreateTestConfigmap(testNamespace, "configmap-1", AppLabels)
	_, err := clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}
	configmap2 := CreateTestConfigmap(testNamespace, "configmap-2", AppLabels)
	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	tests := []struct {
		name          string
		diff          []ResourceInfo
		resourceType  string
		expectedDiff  []ResourceInfo
		expectedError bool
	}{
		{
			name: "Test deletion confirmation",
			diff: []ResourceInfo{
				{Name: configmap1.Name, Reason: "ConfigMap is not used in any pod or container"},
				{Name: configmap2.Name, Reason: "Marked with unused label"},
			},
			resourceType: "ConfigMap",
			expectedDiff: []ResourceInfo{
				{Name: configmap1.Name + "-DELETED", Reason: "ConfigMap is not used in any pod or container"},
				{Name: configmap2.Name + "-DELETED", Reason: "Marked with unused label"},
			},
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deletedDiff, _ := DeleteResource(test.diff, clientset, testNamespace, test.resourceType, true)
			for i, deleted := range deletedDiff {
				if deleted != test.expectedDiff[i] {
					t.Errorf("Expected: %s, Got: %s", test.expectedDiff[i], deleted)
				}
			}
		})
	}
}

func TestDeleteDeleteResourceWithFinalizer(t *testing.T) {
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{Group: "testgroup", Version: "v1", Resource: "TestResource"}
	testResource := CreateTestUnstructered(gvr.Resource, gvr.GroupVersion().String(), testNamespace, "test-resource")
	testResouceInfo := ResourceInfo{Name: testResource.GetName()}
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme, testResource)

	_, err := dynamicClient.Resource(gvr).
		Namespace(testNamespace).
		Create(context.TODO(), testResource, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test resource: %v", err)
	}

	_, err = dynamicClient.
		Resource(gvr).
		Namespace(testNamespace).
		Patch(context.TODO(), "test-resource", types.MergePatchType,
			[]byte(`{"metadata":{"finalizers":["finalizer1", "finalizer2", "finalizer3"]}}`),
			metav1.PatchOptions{})

	if err != nil {
		t.Fatalf("Error patching test resource: %v", err)
	}

	tests := []struct {
		name          string
		diff          []ResourceInfo
		resourceType  string
		expectedDiff  []string
		expectedError bool
	}{
		{
			name:          "Test deletion confirmation",
			diff:          []ResourceInfo{testResouceInfo},
			expectedDiff:  []string{testResource.GetName() + "-DELETED"},
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deletedDiff, _ := DeleteResourceWithFinalizer(test.diff, dynamicClient, testNamespace, gvr, true)

			for i, deleted := range deletedDiff {
				if deleted.Name != test.expectedDiff[i] {
					t.Errorf("Expected: %s, Got: %s", test.expectedDiff[i], deleted)
					resource, err := dynamicClient.Resource(gvr).
						Namespace(testNamespace).
						Get(context.TODO(), deleted.Name, metav1.GetOptions{})
					if err != nil {
						t.Error(err)
					}
					if resource.GetFinalizers() != nil {
						t.Error("Finalizers not patched")
					}
				}
			}

		})
	}
}

func TestFlagDynamicResource(t *testing.T) {
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{Group: "testgroup", Version: "v1", Resource: "TestResource"}
	testResource := CreateTestUnstructered(gvr.Resource, gvr.GroupVersion().String(), testNamespace, "test-resource")
	testResourceWithLabel := CreateTestUnstructered(gvr.Resource, gvr.GroupVersion().String(), testNamespace, "test-resource-with-label")
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme, testResource, testResourceWithLabel)
	testResourceWithLabel.SetLabels(map[string]string{
		"test": "true",
	})

	_, err := dynamicClient.Resource(gvr).
		Namespace(testNamespace).
		Create(context.TODO(), testResource, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test resource: %v", err)
	}
	_, err = dynamicClient.Resource(gvr).
		Namespace(testNamespace).
		Create(context.TODO(), testResourceWithLabel, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test resource with finalizers: %v", err)
	}

	tests := []struct {
		name          string
		gvr           schema.GroupVersionResource
		resourceName  string
		labels        bool
		expectedError bool
	}{
		{
			name:          "Test flagging dynamic resource",
			resourceName:  "test-resource",
			labels:        false,
			expectedError: false,
		},
		{
			name:          "Test flagging dynamic resource with labels",
			resourceName:  "test-resource-with-label",
			labels:        true,
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := FlagDynamicResource(dynamicClient, testNamespace, gvr, test.resourceName)

			if (err != nil) != test.expectedError {
				t.Errorf("Expected error: %v, Got: %v", test.expectedError, err)
			}
			resource, err := dynamicClient.Resource(gvr).
				Namespace(testNamespace).
				Get(context.TODO(), test.resourceName, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}
			if resource.GetLabels()["kor/used"] != "true" {
				t.Errorf("Expected resource flagged as used, Got: %v", resource.GetLabels()["kor/used"])
			}
			if test.labels == true && resource.GetLabels()["test"] != "true" {
				t.Errorf("Resource Lost his labels")
			}
		})
	}
}
