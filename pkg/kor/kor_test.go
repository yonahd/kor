package kor

import (
	"os"
	"sort"
	"testing"
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
	defer os.Remove(configFile.Name())
	if err := os.WriteFile(configFile.Name(), []byte(getFakeConfigContent()), 0666); err != nil {
		t.Error(err)
	}

	originalKCEnv := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", originalKCEnv)
	os.Setenv("KUBECONFIG", configFile.Name())

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
	defer os.Remove(configFile.Name())
	if err := os.WriteFile(configFile.Name(), []byte(getFakeConfigContent()), 0666); err != nil {
		t.Error(err)
	}

	oldKubeServiceHost := os.Getenv("KUBERNETES_SERVICE_HOST")
	oldKubeServicePort := os.Getenv("KUBERNETES_SERVICE_PORT")
	os.Setenv("KUBERNETES_SERVICE_HOST", "127.0.0.1")
	os.Setenv("KUBERNETES_SERVICE_PORT", "443")

	defer func() {
		os.Setenv("KUBERNETES_SERVICE_HOST", oldKubeServiceHost)
		os.Setenv("KUBERNETES_SERVICE_PORT", oldKubeServicePort)
	}()

	kcs := GetKubeClient(configFile.Name())
	if kcs == nil {
		t.Errorf("Expected valid clientSet")
	}
}

func TestNamespacedMessageSuffix(t *testing.T) {
	type args struct {
		namespace string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty string passed",
			args: args{
				namespace: "",
			},
			want: "",
		},
		{
			name: "namespace name passed",
			args: args{
				namespace: "test-ns1",
			},
			want: " in namespace test-ns1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := namespacedMessageSuffix(tt.args.namespace); got != tt.want {
				t.Errorf(
					"namespacedMessageSuffix() = '%v', want '%v'",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestFormatOutput(t *testing.T) {
	type args struct {
		namespace string
		resources []string
		verbose   bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "verbose, empty namespace, empty resource list",
			args: args{
				namespace: "",
				resources: []string{},
				verbose:   true,
			},
			want: "No unused TestType found\n",
		},
		{
			name: "verbose, non empty namespace, empty resource list",
			args: args{
				namespace: "test-ns",
				resources: []string{},
				verbose:   true,
			},
			want: "No unused TestType found in namespace test-ns\n",
		},
		{
			name: "non verbose, empty namespace, empty resource list",
			args: args{
				namespace: "",
				resources: []string{},
				verbose:   false,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatOutput(
				tt.args.namespace,
				tt.args.resources,
				"TestType",
				Opts{Verbose: tt.args.verbose},
			); got != tt.want {
				t.Errorf("FormatOutput() = '%v', want '%v'", got, tt.want)
			}
		})
	}

}
