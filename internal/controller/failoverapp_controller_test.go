package controller

import (
	"testing"

	opsv1alpha1 "github.com/vk86294140-cloud/failover-operator/api/v1alpha1"
)

func TestDesiredDeploymentDefaults(t *testing.T) {
	r := &FailoverAppReconciler{}
	app := &opsv1alpha1.FailoverApp{}
	app.Name = "demo"
	app.Namespace = "default"
	app.Spec.Image = "nginx:1.27"
	// Replicas and Port left at zero so the defaulting path is exercised.

	d := r.desiredDeployment(app)

	if d.Name != "demo-app" {
		t.Fatalf("deployment name = %q, want demo-app", d.Name)
	}
	if d.Spec.Replicas == nil || *d.Spec.Replicas != 2 {
		t.Fatalf("replica default not applied: %v", d.Spec.Replicas)
	}
	c := d.Spec.Template.Spec.Containers
	if len(c) != 1 || c[0].Image != "nginx:1.27" {
		t.Fatalf("container mismatch: %+v", c)
	}
	if len(c[0].Ports) != 1 || c[0].Ports[0].ContainerPort != 8080 {
		t.Fatalf("port default not applied: %+v", c[0].Ports)
	}
	if c[0].ReadinessProbe == nil || c[0].ReadinessProbe.TCPSocket == nil {
		t.Fatal("readiness probe not set")
	}
}

func TestNeedsUpdate(t *testing.T) {
	r := &FailoverAppReconciler{}
	app := &opsv1alpha1.FailoverApp{}
	app.Name = "demo"
	app.Spec.Image = "nginx:1.27"
	app.Spec.Replicas = 3
	base := r.desiredDeployment(app)

	if needsUpdate(base, base.DeepCopy()) {
		t.Fatal("identical deployments should not need an update")
	}

	scaled := base.DeepCopy()
	two := int32(2)
	scaled.Spec.Replicas = &two
	if !needsUpdate(base, scaled) {
		t.Fatal("a replica change should need an update")
	}

	reimaged := base.DeepCopy()
	reimaged.Spec.Template.Spec.Containers[0].Image = "nginx:1.28"
	if !needsUpdate(base, reimaged) {
		t.Fatal("an image change should need an update")
	}
}
