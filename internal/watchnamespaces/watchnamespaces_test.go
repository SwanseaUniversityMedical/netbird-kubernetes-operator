// SPDX-License-Identifier: BSD-3-Clause

package watchnamespaces

import (
	"reflect"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "only separators and spaces", in: " , ,", want: nil},
		{name: "one", in: "ns-a", want: []string{"ns-a"}},
		{name: "several with spaces", in: " ns-a , ns-b,ns-c ", want: []string{"ns-a", "ns-b", "ns-c"}},
		{name: "duplicates removed, first order kept", in: "ns-b,ns-a,ns-b", want: []string{"ns-b", "ns-a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Parse(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCacheOptionsNoNamespacesWatchesAll(t *testing.T) {
	got := CacheOptions(nil)
	if !reflect.DeepEqual(got, cache.Options{}) {
		t.Errorf("CacheOptions(nil) = %#v, want the zero cache.Options", got)
	}
}

func TestCacheOptionsLimitsToNamespaces(t *testing.T) {
	got := CacheOptions([]string{"ns-a", "ns-b"})
	want := map[string]cache.Config{"ns-a": {}, "ns-b": {}}
	if !reflect.DeepEqual(got.DefaultNamespaces, want) {
		t.Errorf("DefaultNamespaces = %#v, want %#v", got.DefaultNamespaces, want)
	}
}

func TestClusterScopedControllersEnabled(t *testing.T) {
	if !ClusterScopedControllersEnabled(nil) {
		t.Error("with no namespace limit, cluster-scoped controllers must run")
	}
	if ClusterScopedControllersEnabled([]string{"ns-a"}) {
		t.Error("with a namespace limit, cluster-scoped controllers must not run")
	}
}
