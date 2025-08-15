package externaldeps

import (
	"testing"
)

func TestResourceReferences(t *testing.T) {
	refs := &ResourceReferences{
		ConfigMaps: []string{"config1", "config2"},
		Secrets:    []string{"secret1"},
		PVCs:       []string{"pvc1", "pvc2"},
	}
	
	// Test that the struct properly holds the expected data
	if len(refs.ConfigMaps) != 2 {
		t.Errorf("expected 2 ConfigMaps, got %d", len(refs.ConfigMaps))
	}
	if refs.ConfigMaps[0] != "config1" || refs.ConfigMaps[1] != "config2" {
		t.Errorf("unexpected ConfigMaps: %v", refs.ConfigMaps)
	}
	
	if len(refs.Secrets) != 1 {
		t.Errorf("expected 1 Secret, got %d", len(refs.Secrets))
	}
	if refs.Secrets[0] != "secret1" {
		t.Errorf("unexpected Secret: %s", refs.Secrets[0])
	}
	
	if len(refs.PVCs) != 2 {
		t.Errorf("expected 2 PVCs, got %d", len(refs.PVCs))
	}
	if refs.PVCs[0] != "pvc1" || refs.PVCs[1] != "pvc2" {
		t.Errorf("unexpected PVCs: %v", refs.PVCs)
	}
}

func TestResourceReferences_Empty(t *testing.T) {
	refs := &ResourceReferences{
		ConfigMaps: make([]string, 0),
		Secrets:    make([]string, 0),
		PVCs:       make([]string, 0),
	}
	
	// Test empty resource references
	if len(refs.ConfigMaps) != 0 {
		t.Errorf("expected 0 ConfigMaps, got %d", len(refs.ConfigMaps))
	}
	if len(refs.Secrets) != 0 {
		t.Errorf("expected 0 Secrets, got %d", len(refs.Secrets))
	}
	if len(refs.PVCs) != 0 {
		t.Errorf("expected 0 PVCs, got %d", len(refs.PVCs))
	}
}
