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

func TestIncludeNamespacesFilter(t *testing.T) {
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
			name: "only include list provided and match",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{
					IncludeNamespaces: []string{"test-ns1", "test-ns2"},
					ExcludeNamespaces: nil,
				},
			},
			want: false,
		},
		{
			name: "only include list provided and no match",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{
					IncludeNamespaces: []string{"test-ns2", "test-ns3"},
					ExcludeNamespaces: nil,
				},
			},
			want: true,
		},
		{
			name: "include list is nil",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{
					IncludeNamespaces: nil,
					ExcludeNamespaces: nil,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IncludeNamespacesFilter(tt.args.object, tt.args.opts); got != tt.want {
				t.Errorf("IncludeNamespacesFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExcludeNamespacesFilter(t *testing.T) {
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
			name: "only exclude list provided and match",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{
					IncludeNamespaces: nil,
					ExcludeNamespaces: []string{"test-ns1", "test-ns2"},
				},
			},
			want: true,
		},
		{
			name: "only exclude list provided and no match",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{
					IncludeNamespaces: nil,
					ExcludeNamespaces: []string{"test-ns2", "test-ns3"},
				},
			},
			want: false,
		},
		{
			name: "exclude list is nil",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{
					IncludeNamespaces: nil,
					ExcludeNamespaces: nil,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExcludeNamespacesFilter(tt.args.object, tt.args.opts); got != tt.want {
				t.Errorf("ExcludeNamespacesFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSystemNamespaceFilter(t *testing.T) {
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
			name: "system namespace - default",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				},
				opts: &Options{},
			},
			want: true,
		},
		{
			name: "system namespace - kube-system",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-system",
					},
				},
				opts: &Options{},
			},
			want: true,
		},
		{
			name: "system namespace - kube-public",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-public",
					},
				},
				opts: &Options{},
			},
			want: true,
		},
		{
			name: "system namespace - kube-node-lease",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kube-node-lease",
					},
				},
				opts: &Options{},
			},
			want: true,
		},
		{
			name: "non system namespace - test-ns1",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SystemNamespaceFilter(tt.args.object, tt.args.opts); got != tt.want {
				t.Errorf("SystemNamespaceFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testHelperFilterTrue(object runtime.Object, filterOpts *Options) bool {
	return true
}

func testHelperFilterFalse(object runtime.Object, filterOpts *Options) bool {
	return false
}

func TestApplyFilters(t *testing.T) {
	type args struct {
		object runtime.Object
		opts   *Options
		funcs  []FilterFunction
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "false,false,false functions",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{},
				funcs: []FilterFunction{
					testHelperFilterFalse,
					testHelperFilterFalse,
					testHelperFilterFalse,
				},
			},
			want: false,
		},
		{
			name: "true,false,true functions",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{},
				funcs: []FilterFunction{
					testHelperFilterTrue,
					testHelperFilterFalse,
					testHelperFilterTrue,
				},
			},
			want: true,
		},
		{
			name: "false,false,true functions",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{},
				funcs: []FilterFunction{
					testHelperFilterFalse,
					testHelperFilterFalse,
					testHelperFilterTrue,
				},
			},
			want: true,
		},
		{
			name: "true,true,true functions",
			args: args{
				object: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns1",
					},
				},
				opts: &Options{},
				funcs: []FilterFunction{
					testHelperFilterTrue,
					testHelperFilterTrue,
					testHelperFilterTrue,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ApplyFilters(tt.args.object, tt.args.opts, tt.args.funcs...); got != tt.want {
				t.Errorf("ApplyFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}
