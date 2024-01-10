package kor

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/utils/strings/slices"
)

func TestCheckFinalizers(t *testing.T) {
	tests := []struct {
		name              string
		finalizers        []string
		deletionTimestamp *metav1.Time
		expectedResult    bool
	}{
		{"EmptyFinalizersAndNilDeletionTimestamp", []string{}, nil, false},
		{"NonEmptyFinalizersAndNilDeletionTimestamp", []string{"finalizer1", "finalizer2"}, nil, false},
		{"EmptyFinalizersAndDeletionTimestamp", []string{}, &metav1.Time{}, false},
		{"NonEmptyFinalizersAndDeletionTimestamp", []string{"finalizer1", "finalizer2"}, &metav1.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckFinalizers(tt.finalizers, tt.deletionTimestamp)
			if result != tt.expectedResult {
				t.Errorf("Expected result %v, but got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestRetrievePendingDeletionResources(t *testing.T) {
	scheme := runtime.NewScheme()

	gvr := schema.GroupVersionResource{Group: "testgroup", Version: "v1", Resource: "testresources"}
	testResource := CreateTestUnstuctered("TestResource", gvr.GroupVersion().String(), testNamespace, "test-resource")
	testResource.SetFinalizers([]string{"test", "test2"})
	testResource.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme, testResource)

	apiResourceLists := []*metav1.APIResourceList{
		{
			GroupVersion: "testgroup/v1",
			APIResources: []metav1.APIResource{
				{
					Name:         "testresources",
					Kind:         "TestResource",
					Verbs:        []string{"list"},
					Namespaced:   true,
					Group:        "testgroup",
					Version:      "v1",
					SingularName: "testresource",
				},
				{
					Name:         "testresourceswithoutlist",
					Kind:         "TestResourceWithoutList",
					Verbs:        []string{"get"},
					Namespaced:   true,
					Group:        "testgroup",
					Version:      "v1",
					SingularName: "testresourcewithoutlist",
				},
			},
		},
		{
			GroupVersion: "bad//api/version",
			APIResources: []metav1.APIResource{},
		},
	}

	tests := []struct {
		name             string
		apiResourceLists []*metav1.APIResourceList
		expectedError    bool
		expectedResult   []string
	}{
		{"resourceInTerminatingState", []*metav1.APIResourceList{apiResourceLists[0]}, false, []string{testResource.GetName()}},
		{"badGVList", []*metav1.APIResourceList{apiResourceLists[1]}, true, []string{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := retrievePendingDeletionResources(test.apiResourceLists, dynamicClient, &FilterOptions{})
			if (err != nil) != test.expectedError {
				t.Errorf("Expected error: %v, Got: %v", test.expectedError, err)
			}
			if deletedResources, ok := result[testNamespace][gvr.GroupVersion().WithResource("testresources")]; ok {
				if !slices.Equal(deletedResources, test.expectedResult) {
					t.Errorf("Expected result: %v, Got: %v", test.expectedResult, deletedResources)
				}
			}
		})
	}
}
