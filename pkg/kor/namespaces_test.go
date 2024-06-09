package kor

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func TestIgnoreResourceType(t *testing.T) {
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

func TestGetGVR(t *testing.T) {
	type args struct {
		name    string
		splitGV []string
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
				name:    "deployments",
				splitGV: []string{},
			},
			want:      nil,
			expectErr: true,
		},
		{
			name: "number of parts 1",
			args: args{
				name: "secrets",
				splitGV: []string{
					"v1",
				},
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
				name: "deployments",
				splitGV: []string{
					"apps",
					"v1",
				},
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
				name: "deployments",
				splitGV: []string{
					"apps",
					"v1",
					"test-deploy01",
				},
			},
			want:      nil,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getGVR(tt.args.name, tt.args.splitGV)
			if (err != nil) != tt.expectErr {
				t.Errorf("getGVR() = expected error: %t, got: '%v'", tt.expectErr, err)
			}
			if got != nil && *got != *tt.want {
				t.Errorf("getGVR() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
func TestIgnorePredefinedResource(t *testing.T) {
	tests := []struct {
		name           string
		gr             GenericResource
		expectedReturn bool
	}{
		{
			name: "configmap kube-root-ca.crt in default",
			gr: GenericResource{
				NamespacedName: types.NamespacedName{
					Name:      "kube-root-ca.crt",
					Namespace: "default",
				},
				GVR: schema.GroupVersionResource{
					Resource: "configmaps",
					Version:  "v1",
				},
			},
			expectedReturn: true,
		},
		{
			name: "configmap kube-root-ca.crt in abc",
			gr: GenericResource{
				NamespacedName: types.NamespacedName{
					Name:      "kube-root-ca.crt",
					Namespace: "abc",
				},
				GVR: schema.GroupVersionResource{
					Resource: "configmaps",
					Version:  "v1",
				},
			},
			expectedReturn: true,
		},
		{
			name: "sa default in default",
			gr: GenericResource{
				NamespacedName: types.NamespacedName{
					Name:      "default",
					Namespace: "default",
				},
				GVR: schema.GroupVersionResource{
					Resource: "serviceaccounts",
					Version:  "v1",
				},
			},
			expectedReturn: true,
		},
		{
			name: "sa default in cde",
			gr: GenericResource{
				NamespacedName: types.NamespacedName{
					Name:      "default",
					Namespace: "cde",
				},
				GVR: schema.GroupVersionResource{
					Resource: "serviceaccounts",
					Version:  "v1",
				},
			},
			expectedReturn: true,
		},
		{
			name: "event in default",
			gr: GenericResource{
				NamespacedName: types.NamespacedName{
					Name:      "test-event",
					Namespace: "default",
				},
				GVR: schema.GroupVersionResource{
					Resource: "events",
				},
			},
			expectedReturn: true,
		},
		{
			name: "event in qqq",
			gr: GenericResource{
				NamespacedName: types.NamespacedName{
					Name:      "test-event",
					Namespace: "qqq",
				},
				GVR: schema.GroupVersionResource{
					Resource: "events",
				},
			},
			expectedReturn: true,
		},
		{
			name: "test-configmap in default",
			gr: GenericResource{
				NamespacedName: types.NamespacedName{
					Name:      "test-configmap",
					Namespace: "default",
				},
				GVR: schema.GroupVersionResource{
					Resource: "configmaps",
					Version:  "v1",
				},
			},
			expectedReturn: false,
		},
		{
			name: "test-serviceaccount in default",
			gr: GenericResource{
				NamespacedName: types.NamespacedName{
					Name:      "test-serviceaccount",
					Namespace: "default",
				},
				GVR: schema.GroupVersionResource{
					Resource: "serviceaccounts",
					Version:  "v1",
				},
			},
			expectedReturn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignorePredefinedResource(tt.gr)
			if got != tt.expectedReturn {
				t.Errorf("ignorePredefinedResource() = %t, want %t", got, tt.expectedReturn)
			}
		})
	}
}
