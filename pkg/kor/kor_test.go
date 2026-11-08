package kor

import (
	"os"
	"reflect"
	"sort"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Sort the slices before comparing
	sort.Strings(a)
	sort.Strings(b)

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestRemoveDuplicatesAndSort(t *testing.T) {
	// Test case 1: Test removing duplicates and sorting the slice
	slice := []string{"b", "a", "c", "b", "a"}
	expected := []string{"a", "b", "c"}
	result := RemoveDuplicatesAndSort(slice)

	if !stringSlicesEqual(result, expected) {
		t.Errorf("RemoveDuplicatesAndSort failed, expected: %v, got: %v", expected, result)
	}

	// Test case 2: Test removing duplicates and sorting an empty slice
	emptySlice := []string{}
	emptyExpected := []string{}
	emptyResult := RemoveDuplicatesAndSort(emptySlice)

	if !stringSlicesEqual(emptyResult, emptyExpected) {
		t.Errorf("RemoveDuplicatesAndSort failed for empty slice, expected: %v, got: %v", emptyExpected, emptyResult)
	}
}

func TestCalculateResourceDifference(t *testing.T) {
	usedResourceNames := []string{"resource1", "resource2", "resource3"}
	allResourceNames := []string{"resource1", "resource2", "resource3", "resource4", "resource5"}

	expectedDifference := []string{"resource4", "resource5"}
	difference := CalculateResourceDifference(usedResourceNames, allResourceNames)

	if len(difference) != len(expectedDifference) {
		t.Errorf("Expected %d difference items, but got %d", len(expectedDifference), len(difference))
	}

	for i, item := range difference {
		if item != expectedDifference[i] {
			t.Errorf("Difference item at index %d should be %s, but got %s", i, expectedDifference[i], item)
		}
	}
}

func getFakeConfigContent() string {
	fakeContent := `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost:8080
    extensions:
    - name: client.authentication.k8s.io/exec
      extension:
        audience: foo
        other: bar
  name: foo-cluster
contexts:
- context:
    cluster: foo-cluster
    namespace: bar
  name: foo-context
current-context: foo-context
kind: Config
`
	return fakeContent
}

func TestGetKubeClientFromEnvVar(t *testing.T) {
	configFile, err := os.CreateTemp("", "kubeconfig-")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err := os.Remove(configFile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	if err := os.WriteFile(configFile.Name(), []byte(getFakeConfigContent()), 0666); err != nil {
		t.Error(err)
	}

	originalKCEnv := os.Getenv("KUBECONFIG")
	defer func() {
		if err := os.Setenv("KUBECONFIG", originalKCEnv); err != nil {
			t.Logf("failed to restore KUBECONFIG: %v", err)
		}
	}()
	if err := os.Setenv("KUBECONFIG", configFile.Name()); err != nil {
		t.Error(err)
		return
	}

	kcs := GetKubeClient("")
	if kcs == nil {
		t.Errorf("Expected valid clientSet")
	}
}

func TestGetKubeClientFromInput(t *testing.T) {
	configFile, err := os.CreateTemp("", "kubeconfig")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err := os.Remove(configFile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	if err := os.WriteFile(configFile.Name(), []byte(getFakeConfigContent()), 0666); err != nil {
		t.Error(err)
	}

	oldKubeServiceHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	oldKubeServicePort := os.Getenv("KUBERNETES_SERVICE_PORT")
	if err := os.Setenv("KUBERNETES_SERVICE_HOST", "127.0.0.1"); err != nil {
		t.Error(err)
		return
	}
	if err := os.Setenv("KUBERNETES_SERVICE_PORT", "443"); err != nil {
		t.Error(err)
		return
	}

	defer func() {
		if err := os.Setenv("KUBERNETES_SERVICE_HOST", oldKubeServiceHost); err != nil {
			t.Logf("failed to restore KUBERNETES_SERVICE_HOST: %v", err)
		}
		if err := os.Setenv("KUBERNETES_SERVICE_PORT", oldKubeServicePort); err != nil {
			t.Logf("failed to restore KUBERNETES_SERVICE_PORT: %v", err)
		}
	}()

	kcs := GetKubeClient(configFile.Name())
	if kcs == nil {
		t.Errorf("Expected valid clientSet")
	}
}

func TestGetClusterName(t *testing.T) {
	configFile, err := os.CreateTemp("", "kubeconfig-")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err := os.Remove(configFile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	if err := os.WriteFile(configFile.Name(), []byte(getFakeConfigContent()), 0666); err != nil {
		t.Error(err)
	}

	clusterName := GetClusterName(configFile.Name())
	if clusterName != "foo-cluster" {
		t.Errorf("Expected %q, got %q", "foo-cluster", clusterName)
	}
}

func TestGetClusterNameInvalidPath(t *testing.T) {
	clusterName := GetClusterName("/nonexistent/kubeconfig")
	if clusterName != "" {
		t.Errorf("Expected empty string, got %q", clusterName)
	}
}

func getFakeExceptions() []ExceptionResource {
	return []ExceptionResource{
		{
			ResourceName: "no-regex",
			Namespace:    "default",
		},
		{
			ResourceName: "with-regex.*",
			Namespace:    "default",
			MatchRegex:   true,
		},
		{
			ResourceName: "with-namespace-regex",
			Namespace:    ".*",
			MatchRegex:   true,
		},
		{
			ResourceName: ".*",
			Namespace:    "with-namespace-regex-prefix-.*",
			MatchRegex:   true,
		},
	}
}

func TestResourceExceptionNoRegex(t *testing.T) {
	exceptions := getFakeExceptions()
	exceptionFound, err := isResourceException("no-regex", "default", exceptions)
	if err != nil {
		t.Error(err)
	}
	if !exceptionFound {
		t.Error("Expected to find exception")
	}
}

func TestResourceExceptionWithRegexInName(t *testing.T) {
	exceptions := getFakeExceptions()
	exceptionFound, err := isResourceException("with-regex-extra-text", "default", exceptions)
	if err != nil {
		t.Error(err)
	}
	if !exceptionFound {
		t.Error("Expected to find exception")
	}
}

func TestResourceExceptionWithRegexInNamespace(t *testing.T) {
	exceptions := getFakeExceptions()
	exceptionFound, err := isResourceException("with-namespace-regex", "default", exceptions)
	if err != nil {
		t.Error(err)
	}
	if !exceptionFound {
		t.Error("Expected to find exception")
	}
}

func TestResourceExceptionWithRegexPrefixInNamespace(t *testing.T) {
	exceptions := getFakeExceptions()
	exceptionFound, err := isResourceException("default", "with-namespace-regex-prefix-extra-text", exceptions)
	if err != nil {
		t.Error(err)
	}
	if !exceptionFound {
		t.Error("Expected to find exception")
	}
}

type fakeDiscoveryWithResources struct {
	*fakediscovery.FakeDiscovery
	resources []*metav1.APIResourceList
}

func (d *fakeDiscoveryWithResources) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return d.resources, nil
}

type fakeClientsetWithDiscovery struct {
	*fake.Clientset
	discovery.DiscoveryInterfaces
}

func (c *fakeClientsetWithDiscovery) Discovery() discovery.DiscoveryInterfaces {
	return c.DiscoveryInterfaces
}

func TestGetResourceKinds(t *testing.T) {
	tests := []struct {
		name          string
		resourceLists []*metav1.APIResourceList
		expectedKinds map[string]ResourceKind
	}{
		{
			name:          "empty discovery",
			expectedKinds: map[string]ResourceKind{},
		},
		{
			name: "resources with singular, plural and short names",
			resourceLists: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{Name: "configmaps", SingularName: "configmap", ShortNames: []string{"cm"}},
						{Name: "pods", SingularName: "pod", ShortNames: []string{"po"}},
						{Name: "services", SingularName: "", ShortNames: []string{"svc"}},
					},
				},
				{
					GroupVersion: "apps/v1",
					APIResources: []metav1.APIResource{
						{Name: "deployments", SingularName: "deployment", ShortNames: []string{"deploy"}},
					},
				},
			},
			expectedKinds: map[string]ResourceKind{
				"configmap":  {Plural: "configmaps", ShortNames: []string{"cm"}},
				"pod":        {Plural: "pods", ShortNames: []string{"po"}},
				"services":   {Plural: "services", ShortNames: []string{"svc"}},
				"deployment": {Plural: "deployments", ShortNames: []string{"deploy"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeDisco := &fakediscovery.FakeDiscovery{
				Fake: &clienttesting.Fake{Resources: test.resourceLists},
			}
			clientset := &fakeClientsetWithDiscovery{
				Clientset: fake.NewClientset(),
				DiscoveryInterfaces: &fakeDiscoveryWithResources{
					FakeDiscovery: fakeDisco,
					resources:     test.resourceLists,
				},
			}

			kinds, err := GetResourceKinds(clientset)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if !reflect.DeepEqual(kinds, test.expectedKinds) {
				t.Errorf("Expected resource kinds %v, got %v", test.expectedKinds, kinds)
			}
		})
	}
}
