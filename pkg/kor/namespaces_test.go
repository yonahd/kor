package kor

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
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
