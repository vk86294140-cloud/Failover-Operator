package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	opsv1alpha1 "github.com/vk86294140-cloud/failover-operator/api/v1alpha1"
)

// failoverAppFinalizer guards deletion so cleanup runs before the object is
// removed from the API server.
const failoverAppFinalizer = "ops.failover.dev/finalizer"

// FailoverAppReconciler reconciles a FailoverApp object toward its desired
// state: it owns a Deployment and reports readiness against MinHealthy.
type FailoverAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// finalize runs teardown for a FailoverApp being deleted. The owned Deployment
// is already cascade-deleted via its owner reference; this hook is the explicit
// place for external teardown (DNS records, cloud resources, drained failover
// targets) so the cleanup path is obvious rather than implicit.
func (r *FailoverAppReconciler) finalize(ctx context.Context, app *opsv1alpha1.FailoverApp) error {
	log.FromContext(ctx).Info("finalizing FailoverApp", "name", app.Name)
	return nil
}

// +kubebuilder:rbac:groups=ops.failover.dev,resources=failoverapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ops.failover.dev,resources=failoverapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ops.failover.dev,resources=failoverapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile drives a single FailoverApp toward its desired state. It is
// level-triggered: it reads the live world, computes what should exist, makes
// it so, and records what it observed in status. The same logic is safe to run
// any number of times for the same object.
func (r *FailoverAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	var app opsv1alpha1.FailoverApp
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		// Ignore not-found: the object was deleted, and the owned Deployment is
		// garbage-collected via the owner reference set below.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion: run cleanup while the finalizer is held, then release it
	// so the API server can remove the object.
	if !app.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&app, failoverAppFinalizer) {
			if err := r.finalize(ctx, &app); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&app, failoverAppFinalizer)
			if err := r.Update(ctx, &app); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure the finalizer is present before we create owned resources, so
	// teardown always has a chance to run.
	if controllerutil.AddFinalizer(&app, failoverAppFinalizer) {
		if err := r.Update(ctx, &app); err != nil {
			return ctrl.Result{}, err
		}
	}

	desired := r.desiredDeployment(&app)
	if err := controllerutil.SetControllerReference(&app, desired, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	var current appsv1.Deployment
	getErr := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, &current)
	switch {
	case apierrors.IsNotFound(getErr):
		l.Info("creating managed Deployment", "deployment", desired.Name)
		if err := r.Create(ctx, desired); err != nil {
			return ctrl.Result{}, err
		}
		current = *desired
	case getErr != nil:
		return ctrl.Result{}, getErr
	default:
		if needsUpdate(&current, desired) {
			current.Spec.Replicas = desired.Spec.Replicas
			current.Spec.Template = desired.Spec.Template
			l.Info("reconciling Deployment drift", "deployment", desired.Name)
			if err := r.Update(ctx, &current); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Refresh from the live Deployment so status reflects reality, not intent.
	if err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, &current); err != nil {
		return ctrl.Result{}, err
	}

	app.Status.AvailableReplicas = current.Status.AvailableReplicas
	app.Status.Ready = current.Status.AvailableReplicas >= app.Spec.MinHealthy

	cond := metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionFalse,
		Reason:             "BelowMinHealthy",
		Message:            "available replicas are below minHealthy",
		ObservedGeneration: app.Generation,
	}
	if app.Status.Ready {
		cond.Status = metav1.ConditionTrue
		cond.Reason = "MinHealthyMet"
		cond.Message = "available replicas are at or above minHealthy"
	}
	meta.SetStatusCondition(&app.Status.Conditions, cond)

	if err := r.Status().Update(ctx, &app); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// desiredDeployment builds the Deployment a FailoverApp should own. Defaults
// mirror the CRD defaults so the function is also correct when called directly.
func (r *FailoverAppReconciler) desiredDeployment(app *opsv1alpha1.FailoverApp) *appsv1.Deployment {
	replicas := app.Spec.Replicas
	if replicas == 0 {
		replicas = 2
	}
	port := app.Spec.Port
	if port == 0 {
		port = 8080
	}
	labels := map[string]string{"failover-app": app.Name}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name + "-app",
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: app.Spec.Image,
						Ports: []corev1.ContainerPort{{ContainerPort: port}},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(port)},
							},
						},
					}},
				},
			},
		},
	}
}

// needsUpdate reports whether the fields this controller owns (replica count
// and container image) have drifted from desired. It deliberately ignores
// fields other controllers or defaulters may set, to avoid update fights.
func needsUpdate(current, desired *appsv1.Deployment) bool {
	if current.Spec.Replicas == nil || desired.Spec.Replicas == nil {
		return true
	}
	if *current.Spec.Replicas != *desired.Spec.Replicas {
		return true
	}
	cc := current.Spec.Template.Spec.Containers
	dc := desired.Spec.Template.Spec.Containers
	if len(cc) != len(dc) {
		return true
	}
	return len(cc) > 0 && cc[0].Image != dc[0].Image
}

// SetupWithManager wires the reconciler to watch FailoverApp objects and the
// Deployments they own, so an edit to either re-triggers reconciliation.
func (r *FailoverAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opsv1alpha1.FailoverApp{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
