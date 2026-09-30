package filters

import (
	"context"
	"reflect"
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewFilterOptions(t *testing.T) {
	opts := NewFilterOptions()

	if opts.OlderThan != "" {
		t.Errorf("OlderThan = %q, want empty", opts.OlderThan)
	}
	if opts.NewerThan != "" {
		t.Errorf("NewerThan = %q, want empty", opts.NewerThan)
	}
	if opts.IncludeLabels != "" {
		t.Errorf("IncludeLabels = %q, want empty", opts.IncludeLabels)
	}
	if len(opts.ExcludeLabels) != 0 {
		t.Errorf("ExcludeLabels = %v, want empty", opts.ExcludeLabels)
	}
}

func TestParseLabels(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "single label",
			input: "app=test",
			want:  map[string]string{"app": "test"},
		},
		{
			name:  "multiple labels",
			input: "a=1,b=2",
			want:  map[string]string{"a": "1", "b": "2"},
		},
		{
			name:    "missing equals sign",
			input:   "app",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLabels(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseLabels() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(map[string]string(got), tt.want) {
				t.Errorf("parseLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name: "empty options",
			opts: &Options{},
		},
		{
			name: "valid options",
			opts: &Options{
				OlderThan:     "1h",
				NewerThan:     "30m",
				ExcludeLabels: []string{"app=test"},
			},
		},
		{
			name:    "invalid exclude label",
			opts:    &Options{ExcludeLabels: []string{"app"}},
			wantErr: true,
		},
		{
			name:    "invalid older-than duration",
			opts:    &Options{OlderThan: "not-a-duration"},
			wantErr: true,
		},
		{
			name:    "negative older-than",
			opts:    &Options{OlderThan: "-1h"},
			wantErr: true,
		},
		{
			name:    "invalid newer-than duration",
			opts:    &Options{NewerThan: "not-a-duration"},
			wantErr: true,
		},
		{
			name:    "negative newer-than",
			opts:    &Options{NewerThan: "-30m"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.opts.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOptionsModify(t *testing.T) {
	tests := []struct {
		name string
		opts *Options
		want []string
	}{
		{
			name: "include labels takes precedence over exclude labels",
			opts: &Options{
				IncludeLabels: "app=test",
				ExcludeLabels: []string{"app!=test"},
			},
			want: nil,
		},
		{
			name: "exclude labels are kept without include labels",
			opts: &Options{
				ExcludeLabels: []string{"app!=test"},
			},
			want: []string{"app!=test"},
		},
		{
			name: "include labels only",
			opts: &Options{
				IncludeLabels: "app=test",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.Modify()
			if !reflect.DeepEqual(tt.opts.ExcludeLabels, tt.want) {
				t.Errorf("Modify() excluded labels = %v, want %v", tt.opts.ExcludeLabels, tt.want)
			}
		})
	}
}

func TestOptionsNamespaces(t *testing.T) {
	clientset := fake.NewClientset()
	for _, name := range []string{"default", "kube-system", "app-ns"} {
		_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("failed to create namespace %s: %v", name, err)
		}
	}

	tests := []struct {
		name string
		opts *Options
		want []string
	}{
		{
			name: "no filters returns all namespaces",
			opts: &Options{},
			want: []string{"app-ns", "default", "kube-system"},
		},
		{
			name: "include namespaces deduplicates and orders",
			opts: &Options{
				IncludeNamespaces: []string{"kube-system", "default", "default"},
			},
			want: []string{"default", "kube-system"},
		},
		{
			name: "include namespaces ignores missing",
			opts: &Options{
				IncludeNamespaces: []string{"default", "does-not-exist"},
			},
			want: []string{"default"},
		},
		{
			name: "exclude namespaces filters from all",
			opts: &Options{
				ExcludeNamespaces: []string{"default"},
			},
			want: []string{"app-ns", "kube-system"},
		},
		{
			name: "include namespaces wins over exclude",
			opts: &Options{
				IncludeNamespaces: []string{"default", "kube-system"},
				ExcludeNamespaces: []string{"kube-system"},
			},
			want: []string{"default", "kube-system"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.Namespaces(clientset)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Namespaces() = %v, want %v", got, tt.want)
			}
		})
	}
}
