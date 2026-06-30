//go:build integration

package controller

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	opsv1alpha1 "github.com/vk86294140-cloud/failover-operator/api/v1alpha1"
)

// This suite runs against a real API server started by envtest. It is gated by
// the `integration` build tag so plain `go test ./...` stays fast and needs no
// cluster; run it with `make test-integration`.

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Integration Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	Expect(opsv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	Expect((&FailoverAppReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)).To(Succeed())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	Expect(testEnv.Stop()).To(Succeed())
})

var _ = Describe("FailoverApp controller", func() {
	It("creates a Deployment for a FailoverApp", func() {
		app := &opsv1alpha1.FailoverApp{
			ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
			Spec: opsv1alpha1.FailoverAppSpec{
				Image:      "nginx:1.27",
				Replicas:   3,
				MinHealthy: 2,
				Port:       80,
			},
		}
		Expect(k8sClient.Create(ctx, app)).To(Succeed())

		key := types.NamespacedName{Name: "demo-app", Namespace: "default"}
		Eventually(func() error {
			var dep appsv1.Deployment
			return k8sClient.Get(ctx, key, &dep)
		}, 10*time.Second, 250*time.Millisecond).Should(Succeed())

		var dep appsv1.Deployment
		Expect(k8sClient.Get(ctx, key, &dep)).To(Succeed())
		Expect(*dep.Spec.Replicas).To(Equal(int32(3)))
		Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal("nginx:1.27"))
	})

	It("sets a finalizer on the FailoverApp", func() {
		key := types.NamespacedName{Name: "demo", Namespace: "default"}
		Eventually(func() bool {
			var app opsv1alpha1.FailoverApp
			if err := k8sClient.Get(ctx, key, &app); err != nil {
				return false
			}
			for _, f := range app.Finalizers {
				if f == "ops.failover.dev/finalizer" {
					return true
				}
			}
			return false
		}, 10*time.Second, 250*time.Millisecond).Should(BeTrue())
	})

	It("removes the FailoverApp cleanly on delete", func() {
		key := types.NamespacedName{Name: "demo", Namespace: "default"}
		var app opsv1alpha1.FailoverApp
		Expect(k8sClient.Get(ctx, key, &app)).To(Succeed())
		Expect(k8sClient.Delete(ctx, &app)).To(Succeed())

		Eventually(func() bool {
			var got opsv1alpha1.FailoverApp
			err := k8sClient.Get(ctx, key, &got)
			return errors.IsNotFound(err)
		}, 10*time.Second, 250*time.Millisecond).Should(BeTrue())
	})
})
