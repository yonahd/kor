package kor

import (
	"testing"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/filters"
)

func createTestCRDs(t *testing.T) (*apiextensionsfake.Clientset, *dynamicfake.FakeDynamicClient) {
	// Create a CRD with a served version
	crdWithServedVersion := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testresources.example.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Scope: apiextensionsv1.ClusterScoped,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "testresources",
				Singular: "testresource",
				Kind:     "TestResource",
			},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  false, // This version is not served
					Storage: false,
				},
				{
					Name:    "v1",
					Served:  true, // This version is served
					Storage: true,
				},
			},
		},
	}

	// Create a CRD with no served version (this should be skipped)
	crdWithNoServedVersion := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "noservedresources.example.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Scope: apiextensionsv1.ClusterScoped,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "noservedresources",
				Singular: "noservedresource",
				Kind:     "NoServedResource",
			},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  false,
					Storage: true,
				},
			},
		},
	}

	// Create a CRD with only the first version not served (testing version selection)
	crdWithFirstVersionNotServed := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "multiresources.example.com",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "example.com",
			Scope: apiextensionsv1.ClusterScoped,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "multiresources",
				Singular: "multiresource",
				Kind:     "MultiResource",
			},
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1alpha1",
					Served:  false,
					Storage: false,
				},
				{
					Name:    "v1beta1",
					Served:  true, // This should be selected
					Storage: false,
				},
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
				},
			},
		},
	}

	apiExtClient := apiextensionsfake.NewClientset(
		crdWithServedVersion,
		crdWithNoServedVersion,
		crdWithFirstVersionNotServed,
	)

	// Create a fake dynamic client with custom list kinds
	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{
		{Group: "example.com", Version: "v1", Resource: "testresources"}:       "TestResourceList",
		{Group: "example.com", Version: "v1beta1", Resource: "multiresources"}: "MultiResourceList",
		{Group: "example.com", Version: "v1", Resource: "multiresources"}:      "MultiResourceList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)

	return apiExtClient, dynamicClient
}

func TestProcessCrds(t *testing.T) {
	apiExtClient, dynamicClient := createTestCRDs(t)

	filterOpts := &filters.Options{}
	unusedCRDs, err := processCrds(apiExtClient, dynamicClient, filterOpts)
	if err != nil {
		t.Fatalf("Error processing CRDs: %v", err)
	}

	// We expect all 3 CRDs to be detected as unused (no instances)
	// But the CRD with no served version should be skipped
	// So we should have 2 unused CRDs
	expectedCount := 2
	if len(unusedCRDs) != expectedCount {
		t.Errorf("Expected %d unused CRDs, got %d", expectedCount, len(unusedCRDs))
		for _, crd := range unusedCRDs {
			t.Logf("Unused CRD: %s - %s", crd.Name, crd.Reason)
		}
	}

	// Verify the CRDs are the ones we expect
	expectedCRDNames := map[string]bool{
		"testresources.example.com":  true,
		"multiresources.example.com": true,
	}

	for _, crd := range unusedCRDs {
		if !expectedCRDNames[crd.Name] {
			t.Errorf("Unexpected CRD in results: %s", crd.Name)
		}
		if crd.Reason != "CRD has no instances" {
			t.Errorf("Expected reason 'CRD has no instances' for %s, got: %s", crd.Name, crd.Reason)
		}
	}
}

func init() {
	// Internal types (REQUIRED for fake client)
	if err := apiextensions.AddToScheme(clientgoscheme.Scheme); err != nil {
		panic(err)
	}

	// External v1 types
	if err := apiextensionsv1.AddToScheme(clientgoscheme.Scheme); err != nil {
		panic(err)
	}
}
