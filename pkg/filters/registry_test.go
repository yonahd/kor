package filters

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func alwaysTrue(object runtime.Object, opts *Options) bool {
	return true
}

func TestNewDefaultRegistry(t *testing.T) {
	r := NewDefaultRegistry()

	if len(r) != 3 {
		t.Errorf("registry has %d filters, want 3", len(r))
	}

	for _, name := range []string{LabelFilterName, AgeFilterName, KorLabelFilterName} {
		if _, ok := r[name]; !ok {
			t.Errorf("default registry is missing filter %q", name)
		}
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewDefaultRegistry()

	if err := r.Register("custom", alwaysTrue); err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}
	if _, ok := r["custom"]; !ok {
		t.Error("Register() did not add the filter")
	}

	if err := r.Register("custom", alwaysTrue); err == nil {
		t.Error("Register() on duplicate name should return an error")
	}
}

func TestRegistryUnregister(t *testing.T) {
	r := NewDefaultRegistry()

	if err := r.Unregister(LabelFilterName); err != nil {
		t.Fatalf("Unregister() returned error: %v", err)
	}
	if _, ok := r[LabelFilterName]; ok {
		t.Error("Unregister() did not remove the filter")
	}

	if err := r.Unregister("does-not-exist"); err == nil {
		t.Error("Unregister() of unknown name should return an error")
	}
}

func TestRegistryMerge(t *testing.T) {
	base := Registry{"a": alwaysTrue}
	other := Registry{
		"b": alwaysTrue,
		"c": alwaysTrue,
	}

	if err := base.Merge(other); err != nil {
		t.Fatalf("Merge() returned error: %v", err)
	}

	if len(base) != 3 {
		t.Errorf("merged registry has %d filters, want 3", len(base))
	}
	for _, name := range []string{"a", "b", "c"} {
		if _, ok := base[name]; !ok {
			t.Errorf("merged registry is missing filter %q", name)
		}
	}
}

func TestRegistryMergeConflict(t *testing.T) {
	base := Registry{"a": LabelFilter}
	other := Registry{"a": alwaysTrue}

	if err := base.Merge(other); err == nil {
		t.Fatal("Merge() with conflicting name should return an error")
	}

	if got := base["a"](nil, &Options{}); got {
		t.Errorf("base filter after failed merge = %v, want %v (original must be unchanged)", got, false)
	}

	if len(base) != 1 {
		t.Errorf("base registry has %d filters, want 1", len(base))
	}
}
