package kor

import (
	"os"
	"sort"
	"strings"
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

func TestGetDefaultKubeConfigPath(t *testing.T) {
	path := getDefaultKubeConfigPath()
	if !strings.Contains(path, ".kube") || !strings.Contains(path, "config") {
		t.Errorf("Expected to find '.kube' and 'config' keywords in path, but got %s", path)
	}
}

func TestLoadOrGetKubeConfigPath_ShouldReadEnvvar(t *testing.T) {
	originalKCEnv := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", originalKCEnv)

	testKCPath := "test/kubeconfig.yaml"
	os.Setenv("KUBECONFIG", testKCPath)
	kcPath := loadOrGetKubeConfigPath("")

	if kcPath != testKCPath {
		t.Errorf("Expected kubeconfig path to be %s, but got %s", testKCPath, kcPath)
	}
}

func TestLoadOrGetKubeConfigPath_ShouldGetDefaultPath(t *testing.T) {
	originalKCEnv := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", originalKCEnv)

	testKCPath := "test/kubeconfig.yaml"
	os.Setenv("KUBECONFIG", "")
	kcPath := loadOrGetKubeConfigPath("")

	if kcPath == testKCPath {
		t.Errorf("Expected kubeconfig path to be different than %s, but got %s", testKCPath, kcPath)
	}

	if !strings.Contains(kcPath, ".kube") || !strings.Contains(kcPath, "config") {
		t.Errorf("Expected to find '.kube' and 'config' keywords in path, but got %s", kcPath)
	}

}

func TestLoadOrGetKubeConfigPath_ShouldReturnSameStringIfNonEmpty(t *testing.T) {
	originalKCEnv := os.Getenv("KUBECONFIG")
	defer os.Setenv("KUBECONFIG", originalKCEnv)

	inputKCPath := "test/inputkc.yaml"

	testKCPath := "test/kubeconfig.yaml"
	os.Setenv("KUBECONFIG", testKCPath)
	kcPath := loadOrGetKubeConfigPath(inputKCPath)

	if kcPath != inputKCPath {
		t.Errorf("Expected kubeconfig path to be %s, but got %s", testKCPath, kcPath)
	}
}
