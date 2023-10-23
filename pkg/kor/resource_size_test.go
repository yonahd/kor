package kor

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func TestGetSize(t *testing.T) {
	tests := []struct {
		name     string
		resource interface{}
		want     *Size
	}{
		{
			name: "ConfigMap",
			resource: &v1.ConfigMap{
				Data: map[string]string{
					"key": "value",
				},
			},
			want: &Size{IntValue: 5},
		},
		{
			name: "Secret",
			resource: &v1.Secret{
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			want: &Size{IntValue: 5},
		},
		{
			name: "Deployment",
			resource: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: pointer.Int32(3),
				},
			},
			want: &Size{IntValue: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSize(tt.resource)
			if err != nil {
				t.Fatalf("GetSize() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
