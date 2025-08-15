package externaldeps

import (
	"context"
	"testing"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// mockScanner implements ExternalResourceScanner for testing
type mockScanner struct {
	name              string
	enabled           bool
	enabledError      error
	scanResult        *ResourceReferences
	scanError         error
	supportedResources []string
}

func (m *mockScanner) GetName() string {
	return m.name
}

func (m *mockScanner) IsEnabled(ctx context.Context, clientset kubernetes.Interface, dynamicClient dynamic.Interface) (bool, error) {
	return m.enabled, m.enabledError
}

func (m *mockScanner) ScanNamespace(ctx context.Context, namespace string, clientset kubernetes.Interface, dynamicClient dynamic.Interface) (*ResourceReferences, error) {
	return m.scanResult, m.scanError
}

func (m *mockScanner) GetSupportedResources() []string {
	return m.supportedResources
}

func TestNewScannerRegistry(t *testing.T) {
	registry := NewScannerRegistry()
	if registry == nil {
		t.Error("expected non-nil registry")
	}
	if registry.scanners == nil {
		t.Error("expected non-nil scanners slice")
	}
	if len(registry.scanners) != 0 {
		t.Errorf("expected empty scanners slice, got %d scanners", len(registry.scanners))
	}
}

func TestScannerRegistry_RegisterScanner(t *testing.T) {
	registry := NewScannerRegistry()
	
	scanner1 := &mockScanner{name: "scanner1"}
	scanner2 := &mockScanner{name: "scanner2"}
	
	registry.RegisterScanner(scanner1)
	if len(registry.scanners) != 1 {
		t.Errorf("expected 1 scanner, got %d", len(registry.scanners))
	}
	
	registry.RegisterScanner(scanner2)
	if len(registry.scanners) != 2 {
		t.Errorf("expected 2 scanners, got %d", len(registry.scanners))
	}
	
	// Verify the scanners are properly registered
	if registry.scanners[0].GetName() != "scanner1" {
		t.Errorf("expected first scanner name 'scanner1', got %q", registry.scanners[0].GetName())
	}
	if registry.scanners[1].GetName() != "scanner2" {
		t.Errorf("expected second scanner name 'scanner2', got %q", registry.scanners[1].GetName())
	}
}

func TestScannerRegistry_ScanNamespace(t *testing.T) {
	registry := NewScannerRegistry()
	clientset := fake.NewSimpleClientset()
	
	// Create mock scanners
	scanner1 := &mockScanner{
		name:    "enabled-scanner",
		enabled: true,
		scanResult: &ResourceReferences{
			ConfigMaps: []string{"config1", "config2"},
			Secrets:    []string{"secret1"},
			PVCs:       []string{},
		},
	}
	
	scanner2 := &mockScanner{
		name:    "disabled-scanner",
		enabled: false,
		scanResult: &ResourceReferences{
			ConfigMaps: []string{"should-not-appear"},
			Secrets:    []string{"should-not-appear"},
			PVCs:       []string{"should-not-appear"},
		},
	}
	
	scanner3 := &mockScanner{
		name:    "another-enabled-scanner",
		enabled: true,
		scanResult: &ResourceReferences{
			ConfigMaps: []string{"config3"},
			Secrets:    []string{"secret2", "secret3"},
			PVCs:       []string{"pvc1"},
		},
	}
	
	registry.RegisterScanner(scanner1)
	registry.RegisterScanner(scanner2)
	registry.RegisterScanner(scanner3)
	
	refs, err := registry.ScanNamespace(context.TODO(), "test-namespace", clientset, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	
	expectedConfigMaps := []string{"config1", "config2", "config3"}
	expectedSecrets := []string{"secret1", "secret2", "secret3"}
	expectedPVCs := []string{"pvc1"}
	
	// Check ConfigMaps
	if len(refs.ConfigMaps) != len(expectedConfigMaps) {
		t.Errorf("expected %d ConfigMaps, got %d: %v", len(expectedConfigMaps), len(refs.ConfigMaps), refs.ConfigMaps)
	}
	for _, expected := range expectedConfigMaps {
		found := false
		for _, actual := range refs.ConfigMaps {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected ConfigMap %q not found in refs", expected)
		}
	}
	
	// Check Secrets
	if len(refs.Secrets) != len(expectedSecrets) {
		t.Errorf("expected %d Secrets, got %d: %v", len(expectedSecrets), len(refs.Secrets), refs.Secrets)
	}
	for _, expected := range expectedSecrets {
		found := false
		for _, actual := range refs.Secrets {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected Secret %q not found in refs", expected)
		}
	}
	
	// Check PVCs
	if len(refs.PVCs) != len(expectedPVCs) {
		t.Errorf("expected %d PVCs, got %d: %v", len(expectedPVCs), len(refs.PVCs), refs.PVCs)
	}
	for _, expected := range expectedPVCs {
		found := false
		for _, actual := range refs.PVCs {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected PVC %q not found in refs", expected)
		}
	}
	
	// Verify disabled scanner results are not included
	for _, configMap := range refs.ConfigMaps {
		if configMap == "should-not-appear" {
			t.Error("disabled scanner results should not be included")
		}
	}
}

func TestScannerRegistry_GetEnabledScanners(t *testing.T) {
	registry := NewScannerRegistry()
	clientset := fake.NewSimpleClientset()
	
	scanner1 := &mockScanner{name: "enabled1", enabled: true}
	scanner2 := &mockScanner{name: "disabled1", enabled: false}
	scanner3 := &mockScanner{name: "enabled2", enabled: true}
	scanner4 := &mockScanner{name: "disabled2", enabled: false}
	
	registry.RegisterScanner(scanner1)
	registry.RegisterScanner(scanner2)
	registry.RegisterScanner(scanner3)
	registry.RegisterScanner(scanner4)
	
	enabled, err := registry.GetEnabledScanners(context.TODO(), clientset, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	
	if len(enabled) != 2 {
		t.Errorf("expected 2 enabled scanners, got %d", len(enabled))
		return
	}
	
	// Check that only enabled scanners are returned
	enabledNames := make(map[string]bool)
	for _, scanner := range enabled {
		enabledNames[scanner.GetName()] = true
	}
	
	if !enabledNames["enabled1"] {
		t.Error("expected 'enabled1' scanner to be in enabled list")
	}
	if !enabledNames["enabled2"] {
		t.Error("expected 'enabled2' scanner to be in enabled list")
	}
	if enabledNames["disabled1"] {
		t.Error("'disabled1' scanner should not be in enabled list")
	}
	if enabledNames["disabled2"] {
		t.Error("'disabled2' scanner should not be in enabled list")
	}
}

func TestGetGlobalRegistry(t *testing.T) {
	// Test that GetGlobalRegistry returns the same instance
	registry1 := GetGlobalRegistry()
	registry2 := GetGlobalRegistry()
	
	if registry1 != registry2 {
		t.Error("GetGlobalRegistry should return the same instance (singleton pattern)")
	}
	
	// Test that the registry is properly initialized with default scanners
	if len(registry1.scanners) == 0 {
		t.Error("expected global registry to have default scanners registered")
	}
	
	// Check that WorkflowTemplate scanner is registered
	found := false
	for _, scanner := range registry1.scanners {
		if scanner.GetName() == "Argo Workflows WorkflowTemplate" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected WorkflowTemplate scanner to be registered in global registry")
	}
}
