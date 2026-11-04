// SPDX-License-Identifier: BSD-3-Clause

// Package watchnamespaces limits the operator's cache to a set of namespaces, so that more than
// one operator install, each with its own management URL, can run in one cluster.
package watchnamespaces

import (
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/cache"
)

// Parse splits a comma-separated namespace list. Spaces and empty entries are dropped, and
// duplicates are removed. It returns nil when no namespace is given.
func Parse(s string) []string {
	var namespaces []string
	seen := map[string]bool{}
	for _, ns := range strings.Split(s, ",") {
		ns = strings.TrimSpace(ns)
		if ns == "" || seen[ns] {
			continue
		}
		seen[ns] = true
		namespaces = append(namespaces, ns)
	}
	return namespaces
}

// CacheOptions returns cache options that watch only the given namespaces. With no namespaces,
// it returns the zero value, which watches all namespaces.
func CacheOptions(namespaces []string) cache.Options {
	if len(namespaces) == 0 {
		return cache.Options{}
	}
	defaultNamespaces := make(map[string]cache.Config, len(namespaces))
	for _, ns := range namespaces {
		defaultNamespaces[ns] = cache.Config{}
	}
	return cache.Options{DefaultNamespaces: defaultNamespaces}
}

// ClusterScopedControllersEnabled reports whether controllers for cluster-scoped kinds may run.
// With a namespace limit they may not: every install in the cluster would reconcile the same
// cluster-scoped objects against its own management URL.
func ClusterScopedControllersEnabled(namespaces []string) bool {
	return len(namespaces) == 0
}
