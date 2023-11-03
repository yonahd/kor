package kor

import (
	"testing"

	"k8s.io/client-go/kubernetes/fake"
)

func TestDeleteResource(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	tests := []struct {
		name          string
		diff          []string
		resourceType  string
		expectedDiff  []string
		expectedError bool
	}{
		{
			name:          "Test deletion confirmation",
			diff:          []string{"resource1", "resource2"},
			resourceType:  "ConfigMap",
			expectedDiff:  []string{"resource1-DELETED", "resource2"},
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deletedDiff, _ := DeleteResource(test.diff, clientset, "namespace", test.resourceType, true)

			for i, deleted := range deletedDiff {
				if deleted != test.expectedDiff[i] {
					t.Errorf("Expected: %s, Got: %s", test.expectedDiff[i], deleted)
				}
			}
		})
	}
}
