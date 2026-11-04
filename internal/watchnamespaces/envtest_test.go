// SPDX-License-Identifier: BSD-3-Clause

package watchnamespaces

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	nbv1alpha1 "github.com/netbirdio/kubernetes-operator/api/v1alpha1"
)

// seenReconciler records the namespace of every NetworkRouter request it gets.
type seenReconciler struct {
	mu   sync.Mutex
	seen map[string]bool
}

func (r *seenReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen[req.Namespace] = true
	return ctrl.Result{}, nil
}

func (r *seenReconciler) saw(ns string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seen[ns]
}

func startEnv(t *testing.T) (*rest.Config, *runtime.Scheme) {
	t.Helper()
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		entries, err := os.ReadDir(filepath.Join("..", "..", "bin", "k8s"))
		if err != nil || len(entries) == 0 {
			t.Skip("no envtest binaries: run make setup-envtest")
		}
		t.Setenv("KUBEBUILDER_ASSETS", filepath.Join("..", "..", "bin", "k8s", entries[0].Name()))
	}
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "charts", "netbird-operator", "crds")},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	t.Cleanup(func() { _ = env.Stop() })

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := nbv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return cfg, scheme
}

func createRouter(t *testing.T, c client.Client, ns string) {
	t.Helper()
	ctx := context.Background()
	if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}); err != nil {
		t.Fatalf("create namespace %s: %v", ns, err)
	}
	router := &nbv1alpha1.NetworkRouter{
		ObjectMeta: metav1.ObjectMeta{Name: "router", Namespace: ns},
		Spec:       nbv1alpha1.NetworkRouterSpec{DNSZoneRef: nbv1alpha1.DNSZoneReference{Name: "example.com"}},
	}
	if err := c.Create(ctx, router); err != nil {
		t.Fatalf("create NetworkRouter in %s: %v", ns, err)
	}
}

// runManager starts a manager with the given namespaces and returns what it reconciled after the
// routers in watched and other exist.
func runManager(t *testing.T, cfg *rest.Config, scheme *runtime.Scheme, namespaces []string, watched, other string) *seenReconciler {
	t.Helper()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
		Cache:   CacheOptions(namespaces),
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	r := &seenReconciler{seen: map[string]bool{}}
	if err := ctrl.NewControllerManagedBy(mgr).
		Named("seen-" + watched).
		For(&nbv1alpha1.NetworkRouter{}).
		Complete(r); err != nil {
		t.Fatalf("new controller: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- mgr.Start(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("manager: %v", err)
		}
	})

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}
	createRouter(t, c, watched)
	createRouter(t, c, other)

	deadline := time.Now().Add(30 * time.Second)
	for !r.saw(watched) {
		if time.Now().After(deadline) {
			t.Fatalf("the router in %s was not reconciled", watched)
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Give the other namespace's event time to arrive if the cache watches it.
	time.Sleep(3 * time.Second)
	return r
}

func TestManagerReconcilesOnlyWatchedNamespaces(t *testing.T) {
	cfg, scheme := startEnv(t)

	t.Run("no namespaces: both routers are reconciled", func(t *testing.T) {
		r := runManager(t, cfg, scheme, nil, "all-a", "all-b")
		if !r.saw("all-b") {
			t.Fatal("control failed: the router in all-b was not reconciled with no namespace limit")
		}
	})

	t.Run("one namespace: the other router is not reconciled", func(t *testing.T) {
		r := runManager(t, cfg, scheme, []string{"only-a"}, "only-a", "only-b")
		if r.saw("only-b") {
			t.Fatal("the router in only-b was reconciled, but only only-a is watched")
		}
	})
}
