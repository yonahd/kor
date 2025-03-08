package kor

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/yonahd/kor/pkg/filters"
)

func Test_namespaces_IgnoreResourceType(t *testing.T) {
	type args struct {
		resource        string
		ignoreResources []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "non matching resource",
			args: args{
				resource: "pods",
				ignoreResources: []string{
					"configmaps",
					"secrets",
				},
			},
			want: false,
		},
		{
			name: "matching resource",
			args: args{
				resource: "secrets",
				ignoreResources: []string{
					"configmaps",
					"secrets",
				},
			},
			want: true,
		},
		{
			name: "empty resource ignore list",
			args: args{
				resource:        "secrets",
				ignoreResources: []string{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ignoreResourceType(tt.args.resource, tt.args.ignoreResources); got != tt.want {
				t.Errorf("ignoreResourceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_namespaces_GetGVR(t *testing.T) {
	type args struct {
		groupVersion string
		name         string
	}
	tests := []struct {
		name      string
		args      args
		want      *schema.GroupVersionResource
		expectErr bool
	}{
		{
			name: "number of parts 0 - expect error",
			args: args{
				groupVersion: "",
				name:         "deployments",
			},
			want:      nil,
			expectErr: true,
		},
		{
			name: "number of parts 1",
			args: args{
				groupVersion: "v1",
				name:         "secrets",
			},
			want: &schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "secrets",
			},
			expectErr: false,
		},
		{
			name: "number of parts 2",
			args: args{
				groupVersion: "apps/v1",
				name:         "deployments",
			},
			want: &schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
			expectErr: false,
		},
		{
			name: "number of parts 4 - expect error",
			args: args{
				groupVersion: "apps/v1/test-deploy01",
				name:         "deployments",
			},
			want:      nil,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getGVR(tt.args.groupVersion, tt.args.name)
			if (err != nil) != tt.expectErr {
				t.Errorf("getGVR() = expected error: %t, got: '%v'", tt.expectErr, err)
			}
			if got != nil && *got != *tt.want {
				t.Errorf("getGVR() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_namespaces_IsNamespaceNotEmpty(t *testing.T) {
	tests := []struct {
		name           string
		gvr            *schema.GroupVersionResource
		objects        *unstructured.UnstructuredList
		filterOpts     *filters.Options
		expectedReturn bool
	}{
		{
			name: "deployment exists, ignoring secrets and configmaps",
			gvr: &schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
			objects: &unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "apps/v1",
							"kind":       "Deployment",
							"metadata": map[string]interface{}{
								"name":      "test-deployment",
								"namespace": "default",
							},
						},
					},
				},
			},
			filterOpts: &filters.Options{
				IgnoreResourceTypes: []string{"configmaps", "secrets"},
			},
			expectedReturn: true,
		},
		{
			name: "deployment exists, ignoring deployments",
			gvr: &schema.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: "deployments",
			},
			objects: &unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "apps/v1",
							"kind":       "Deployment",
							"metadata": map[string]interface{}{
								"name":      "test-deployment",
								"namespace": "default",
							},
						},
					},
				},
			},
			filterOpts: &filters.Options{
				IgnoreResourceTypes: []string{"deployments"},
			},
			expectedReturn: false,
		},
		{
			name: "event exists but ignored, ignoring deployments",
			gvr: &schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "events",
			},
			objects: &unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "Event",
							"metadata": map[string]interface{}{
								"name":      "pod-event",
								"namespace": "abc",
							},
						},
					},
				},
			},
			filterOpts: &filters.Options{
				IgnoreResourceTypes: []string{"deployments"},
			},
			expectedReturn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNamespaceNotEmpty(tt.gvr, tt.objects, tt.filterOpts)
			if got != tt.expectedReturn {
				t.Errorf("Expected namespace to be not empty (%t), but result is %t", tt.expectedReturn, got)
			}
		})
	}
}
