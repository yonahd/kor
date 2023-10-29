package kor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHasIncludedAge(t *testing.T) {
	tests := []struct {
		name         string
		creationTime time.Time // resource creation time
		opts         *FilterOptions
		want         bool
	}{
		{
			name:         "The resource is not older than 20 minutes",
			creationTime: metav1.Now().Time,
			opts:         &FilterOptions{NewerThan: "20m"},
			want:         true,
		}, {
			name:         "The resource age is more than 10 second",
			creationTime: metav1.Now().Add(-12 * time.Second),
			opts:         &FilterOptions{OlderThan: "10s"},
			want:         true,
		}, {
			name:         "Two flags are provided",
			creationTime: metav1.Now().Time,
			opts:         &FilterOptions{OlderThan: "20m", NewerThan: "10m"},
			want:         false,
		},
	}

	for _, tt := range tests {
		got, _ := HasIncludedAge(
			metav1.NewTime(tt.creationTime),
			tt.opts,
		)
		assert.Equal(t, tt.want, got)
	}

}

func TestHasExcludedLabel(t *testing.T) {
	tests := []struct {
		resourcelabels  map[string]string
		excludeSelector string
		want            bool
	}{
		{
			resourcelabels:  map[string]string{"key1": "val1", "key2": "val2", "key3": "val3"},
			excludeSelector: "key2=val2",
			want:            true,
		},
		{
			resourcelabels:  map[string]string{"key1": "val1", "key2": "val2", "key3": "val3"},
			excludeSelector: "",
			want:            false,
		},
		{
			resourcelabels:  map[string]string{"key1": "val1", "key2": "val2", "key3": "val3"},
			excludeSelector: "key4=val1",
			want:            false,
		},
		{
			resourcelabels:  map[string]string{"key1": "val1", "key2": "val2", "key3": "val3"},
			excludeSelector: "key1=val5",
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := HasExcludedLabel(tt.resourcelabels, tt.excludeSelector)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasIncludedSize(t *testing.T) {
	resource1 := &v1.ConfigMap{Data: map[string]string{"key1": "val1"}}

	tests := []struct {
		resource interface{}
		opts     *FilterOptions
		want     bool
	}{
		{
			resource: resource1,
			opts:     &FilterOptions{MaxSize: uint64(5)},
			want:     true,
		}, {
			resource: resource1,
			opts:     &FilterOptions{MinSize: uint64(3)},
			want:     true,
		}, {
			resource: resource1,
			opts:     &FilterOptions{MinSize: uint64(3), MaxSize: uint64(5)},
			want:     true,
		}, {
			resource: resource1,
			opts:     &FilterOptions{MinSize: uint64(4), MaxSize: uint64(5)},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := HasIncludedSize(tt.resource, tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
