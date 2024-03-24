package filters

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestLabelFilter(t *testing.T) {
	type args struct {
		object runtime.Object
		opts   *Options
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
				"app": "test",
			},
		},
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "don't have exclude label",
			args: args{
				object: node,
				opts: &Options{
					ExcludeLabels: []string{"foo=barbar"},
				},
			},
			want: false,
		},
		{
			name: "have exclude label",
			args: args{
				object: node,
				opts: &Options{
					ExcludeLabels: []string{"foo=bar"},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		got := LabelFilter(tt.args.object, tt.args.opts)
		if got != tt.want {
			t.Errorf("%s LabelFilter() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestAgeFilter(t *testing.T) {
	type args struct {
		object runtime.Object
		opts   *Options
	}
	now := time.Now()
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not older than 3h",
			args: args{
				object: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{Time: now},
					},
				},
				opts: &Options{
					NewerThan: "3h",
				},
			},
			want: false,
		},
		{
			name: "more than 3h",
			args: args{
				object: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{Time: now.Add(-4 * time.Hour)},
					},
				},
				opts: &Options{
					OlderThan: "3h",
				},
			},
			want: false,
		},
		{
			name: "two flags are provided",
			args: args{
				object: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{Time: now.Add(-4 * time.Hour)},
					},
				},
				opts: &Options{
					NewerThan: "3h",
					OlderThan: "2h",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		got := AgeFilter(tt.args.object, tt.args.opts)
		if got != tt.want {
			t.Errorf("%s HasIncludedAge() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestKorLabelFilter(t *testing.T) {
	type args struct {
		object runtime.Object
		opts   *Options
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "don't have kor/used label",
			args: args{
				object: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "have kor/used label true",
			args: args{
				object: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kor/used": "true",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "have kor/used label false",
			args: args{
				object: &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"kor/used": "false",
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KorLabelFilter(tt.args.object, tt.args.opts); got != tt.want {
				t.Errorf("KorLabelFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
