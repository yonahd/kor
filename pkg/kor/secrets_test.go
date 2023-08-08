package kor

import (
	"testing"
)

func TestCheckExceptions(t *testing.T) {
	tests := []struct {
		SecretName string
		Namespace  string
		Expected   bool
	}{
		{"sh.helm.release.v1.secret1", "default", true},
		{"sh.helm.release.v1.secret2", "namespace1", true},
		{"sh.helm.release.v2.secret", "default", false},
		{"other.unknown.secret", "default", false},
	}

	for _, test := range tests {
		actual := checkExceptions(test.SecretName, test.Namespace)
		if actual != test.Expected {
			t.Errorf("For SecretName=%s, Namespace=%s, expected %v, but got %v", test.SecretName, test.Namespace, test.Expected, actual)
		}
	}
}
