package kor

import (
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
