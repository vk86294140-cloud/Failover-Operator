package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FailoverAppSpec is the desired state of a FailoverApp.
type FailoverAppSpec struct {
	// Image is the container image the managed Deployment runs.
	Image string `json:"image"`

	// Replicas is the desired number of pods.
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas,omitempty"`

	// MinHealthy is the number of available pods required before the
	// FailoverApp reports Ready. It models the "must hold on the worst day"
	// floor: below this, the app is considered unhealthy.
	// +kubebuilder:validation:Minimum=1
	MinHealthy int32 `json:"minHealthy"`

	// Port is the container port the app listens on. It is also used for the
	// readiness probe on the managed pods.
	// +kubebuilder:default=8080
	Port int32 `json:"port,omitempty"`
}

// FailoverAppStatus is the observed state of a FailoverApp.
type FailoverAppStatus struct {
	// AvailableReplicas mirrors the managed Deployment's available replicas.
	AvailableReplicas int32 `json:"availableReplicas"`

	// Ready is true when AvailableReplicas >= MinHealthy.
	Ready bool `json:"ready"`

	// Conditions hold the latest observations of the FailoverApp state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// FailoverApp ensures a Deployment exists for a workload and reports whether it
// is holding at or above a minimum-healthy replica floor.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=fa
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Available",type=integer,JSONPath=`.status.availableReplicas`
// +kubebuilder:printcolumn:name="Desired",type=integer,JSONPath=`.spec.replicas`
type FailoverApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FailoverAppSpec   `json:"spec,omitempty"`
	Status FailoverAppStatus `json:"status,omitempty"`
}

// FailoverAppList contains a list of FailoverApp.
//
// +kubebuilder:object:root=true
type FailoverAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FailoverApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FailoverApp{}, &FailoverAppList{})
}
