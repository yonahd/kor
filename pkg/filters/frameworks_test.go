package filters

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func testNode(labels map[string]string) runtime.Object {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
	}
}

func TestNewNormalFramework(t *testing.T) {
	f := NewNormalFramework(NewDefaultRegistry())

	nf, ok := f.(*normalFramework)
	if !ok {
		t.Fatalf("NewNormalFramework() returned %T, want *normalFramework", f)
	}

	if len(nf.registry) != 3 {
		t.Errorf("registry has %d filters, want 3", len(nf.registry))
	}

	for _, name := range []string{LabelFilterName, AgeFilterName, KorLabelFilterName} {
		if _, ok := nf.registry[name]; !ok {
			t.Errorf("default registry is missing filter %q", name)
		}
	}
}

func TestNormalFrameworkRun(t *testing.T) {
	cleanNode := testNode(nil)
	usedNode := testNode(map[string]string{"kor/used": "true"})
	matchedNode := testNode(map[string]string{"app": "test"})

	tests := []struct {
		name    string
		object  runtime.Object
		opts    *Options
		disable []string
		want    bool
	}{
		{
			name:   "clean object is not filtered",
			object: cleanNode,
			opts:   &Options{},
			want:   false,
		},
		{
			name:   "kor/used=true is filtered",
			object: usedNode,
			opts:   &Options{},
			want:   true,
		},
		{
			name:   "exclude label match is filtered",
			object: matchedNode,
			opts:   &Options{ExcludeLabels: []string{"app=test"}},
			want:   true,
		},
		{
			name:    "disabling kor label filter stops filtering",
			object:  usedNode,
			opts:    &Options{},
			disable: []string{KorLabelFilterName},
			want:    false,
		},
		{
			name:    "disabling unknown filter has no effect",
			object:  usedNode,
			opts:    &Options{},
			disable: []string{"nonexistent"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewNormalFramework(NewDefaultRegistry()).SetObject(tt.object)
			got, err := f.Run(tt.opts, tt.disable...)
			if err != nil {
				t.Fatalf("Run() returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalFrameworkRunAgeFilter(t *testing.T) {
	tooOldNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-4 * time.Hour)},
		},
	}
	recentNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.Time{Time: time.Now()},
		},
	}

	tests := []struct {
		name   string
		object runtime.Object
		opts   *Options
		want   bool
	}{
		{
			name:   "resource outside newer-than range is filtered",
			object: tooOldNode,
			opts:   &Options{NewerThan: "3h"},
			want:   true,
		},
		{
			name:   "resource inside newer-than range is kept",
			object: recentNode,
			opts:   &Options{NewerThan: "3h"},
			want:   false,
		},
		{
			name:   "resource older than older-than is kept",
			object: tooOldNode,
			opts:   &Options{OlderThan: "3h"},
			want:   false,
		},
		{
			name:   "resource not older than older-than is filtered",
			object: recentNode,
			opts:   &Options{OlderThan: "3h"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewNormalFramework(NewDefaultRegistry()).SetObject(tt.object)
			got, err := f.Run(tt.opts)
			if err != nil {
				t.Fatalf("Run() returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunFilter(t *testing.T) {
	f := NewNormalFramework(NewDefaultRegistry()).SetObject(
		testNode(map[string]string{"kor/used": "true"}),
	)

	tests := []struct {
		name string
		f    string
		want bool
	}{
		{
			name: "existing filter",
			f:    KorLabelFilterName,
			want: true,
		},
		{
			name: "unknown filter returns true",
			f:    "unknown",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.RunFilter(tt.f, &Options{})
			if err != nil {
				t.Fatalf("RunFilter() returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("RunFilter(%q) = %v, want %v", tt.f, got, tt.want)
			}
		})
	}
}

func TestAddFilter(t *testing.T) {
	base := NewNormalFramework(NewDefaultRegistry()).SetObject(testNode(nil))
	out := base.AddFilter("always-true", func(object runtime.Object, opts *Options) bool {
		return true
	})

	got, err := out.Run(&Options{})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if !got {
		t.Errorf("Run() on framework with added filter = %v, want true", got)
	}

	got, err = base.Run(&Options{})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if got {
		t.Errorf("Run() on original framework = %v, want false (AddFilter must not mutate receiver)", got)
	}
}

func TestAddFilterDuplicateName(t *testing.T) {
	base := NewNormalFramework(NewDefaultRegistry()).SetObject(testNode(nil))
	out := base.AddFilter(KorLabelFilterName, func(object runtime.Object, opts *Options) bool {
		return true
	})

	nf, ok := out.(*normalFramework)
	if !ok {
		t.Fatalf("out is %T, want *normalFramework", out)
	}

	got, err := out.Run(&Options{})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(nf.registry) != 3 {
		t.Errorf("registry has %d filters, want 3 (duplicate Register must be ignored)", len(nf.registry))
	}
	if got {
		t.Errorf("Run() = %v, want false (duplicate filter must not replace existing)", got)
	}
}

func TestSetObject(t *testing.T) {
	f := NewNormalFramework(NewDefaultRegistry())

	clean := f.SetObject(testNode(nil))
	used := f.SetObject(testNode(map[string]string{"kor/used": "true"}))

	got, err := clean.Run(&Options{})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if got {
		t.Errorf("Run() on clean object = %v, want false", got)
	}

	got, err = used.Run(&Options{})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if !got {
		t.Errorf("Run() on kor/used=true object = %v, want true", got)
	}

	got, err = f.Run(&Options{})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if got {
		t.Errorf("Run() on unmodified receiver = %v, want false (SetObject must not mutate receiver)", got)
	}
}

func TestSetRegistry(t *testing.T) {
	f := NewNormalFramework(NewDefaultRegistry())
	custom := Registry{
		"always-true": func(object runtime.Object, opts *Options) bool { return true },
	}

	out := f.SetRegistry(custom)

	got, err := out.Run(&Options{})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if !got {
		t.Errorf("Run() with custom registry = %v, want true", got)
	}

	got, err = f.Run(&Options{})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if got {
		t.Errorf("Run() on unmodified receiver = %v, want false (SetRegistry must not mutate receiver)", got)
	}
}

func TestNormalFrameworkDeepCopy(t *testing.T) {
	nf := &normalFramework{registry: NewDefaultRegistry()}
	cp := nf.DeepCopy()

	if cp == nf {
		t.Error("DeepCopy() returned the same pointer")
	}

	delete(cp.registry, LabelFilterName)
	if len(cp.registry) != 2 {
		t.Errorf("copy registry has %d filters, want 2", len(cp.registry))
	}
	if len(nf.registry) != 3 {
		t.Errorf("source registry has %d filters, want 3 (DeepCopy must not share registry)", len(nf.registry))
	}
}

func TestNormalFrameworkDeepCopyNilRegistry(t *testing.T) {
	nf := &normalFramework{}
	cp := nf.DeepCopy()

	if cp.registry != nil {
		t.Errorf("copy registry = %v, want nil", cp.registry)
	}
}

func TestNormalFrameworkDeepCopyInto(t *testing.T) {
	src := &normalFramework{registry: Registry{"a": LabelFilter}}
	dst := &normalFramework{registry: Registry{"b": AgeFilter}}

	src.DeepCopyInto(dst)

	if _, ok := dst.registry["a"]; !ok {
		t.Errorf("dst registry is missing filter %q", "a")
	}
	if _, ok := dst.registry["b"]; ok {
		t.Errorf("dst registry should not retain filter %q after DeepCopyInto", "b")
	}

	if got := dst.registry["a"](testNode(map[string]string{"app": "test"}), &Options{ExcludeLabels: []string{"app=test"}}); !got {
		t.Error("dst registry filter must behave as the source LabelFilter")
	}

	if got := dst.registry["a"](testNode(map[string]string{"app": "test"}), &Options{}); got {
		t.Errorf("dst registry filter = %v, want false (DeepCopyInto must copy the exact filter)", got)
	}
}

func TestIsIn(t *testing.T) {
	tests := []struct {
		name    string
		item    string
		disable []string
		want    bool
	}{
		{name: "present", item: "a", disable: []string{"b", "a"}, want: true},
		{name: "absent", item: "c", disable: []string{"b", "a"}, want: false},
		{name: "nil disable", item: "a", disable: nil, want: false},
		{name: "empty disable", item: "a", disable: []string{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIn(tt.item, tt.disable); got != tt.want {
				t.Errorf("isIn(%q, %v) = %v, want %v", tt.item, tt.disable, got, tt.want)
			}
		})
	}
}
